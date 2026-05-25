package ui

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/astro"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/catalog"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/config"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ephemeris"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/logging"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/server"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/slew"
	"github.com/zachsis/skyquest-xtg-synscan-planetarium/internal/ui/panels"
)

// MainApp holds the application window, navigation, and panels.
type MainApp struct {
	fyneApp      fyne.App
	window       fyne.Window
	config       *config.Config
	nav          *widget.List
	content      *fyne.Container
	panels       []Panel
	StatusPanel  *StatusPanel // exposed for PositionProvider consumers
}

// NewMainApp creates a new MainApp.
func NewMainApp(a fyne.App, w fyne.Window, cfg *config.Config) *MainApp {
	return &MainApp{
		fyneApp: a,
		window:  w,
		config:  cfg,
	}
}

// Setup initializes the navigation panels and window layout.
func (m *MainApp) Setup() {
	settingsPanel := panels.NewSettingsPanel(m.config, nil)
	astroSvc := astro.NewAstroService(m.config)
	statusPanel := NewStatusPanel(m.config, astroSvc)
	m.StatusPanel = statusPanel

	cat, err := catalog.NewCatalog(astroSvc)
	if err != nil {
		log.Printf("warning: star catalog failed to load: %v", err)
	}

	var engine *ephemeris.EphemerisEngine
	if cat != nil {
		var engErr error
		engine, engErr = ephemeris.NewEphemerisEngine(astroSvc)
		if engErr != nil {
			log.Printf("warning: ephemeris engine failed to start: %v", engErr)
		} else {
			cat.Registry.RegisterDynamic(engine)
			engine.Start(context.Background())
		}
	}

	slewSvc := slew.NewGoToService(statusPanel.Controller())
	trackingPanel := NewTrackingPanel(statusPanel, statusPanel.Controller())
	gotoPanel := NewGoToPanel(statusPanel, slewSvc, astroSvc, m.config)

	// --- Stellarium TCP server ---
	m.config.RLock()
	stelPort := m.config.StellariumPort
	stelIntervalMs := m.config.StellariumIntervalMs
	stelEnabled := m.config.StellariumEnabled
	m.config.RUnlock()

	stelInterval := time.Duration(stelIntervalMs) * time.Millisecond
	stelSrv := server.NewStellariumServer(stelPort, stelInterval, slewSvc)
	stelSrv.SetPositionProvider(statusPanel)

	// --- Observation logging ---
	var logStore logging.LogStore
	var activeSessionID int64
	var unsubAutoLog func()

	if cfgDir, err := config.ConfigDir(); err != nil {
		log.Printf("warning: cannot determine config dir for log db: %v", err)
	} else {
		if err := os.MkdirAll(cfgDir, 0700); err != nil {
			log.Printf("warning: cannot create config dir: %v", err)
		}
		dbPath := filepath.Join(cfgDir, "observations.db")
		if store, err := logging.OpenSQLiteStore(dbPath); err != nil {
			log.Printf("warning: observation log db failed to open: %v", err)
		} else {
			logStore = store

			sess := &logging.Session{
				StartedAt:    time.Now().UTC(),
				ObserverLat:  m.config.GetLatitude(),
				ObserverLon:  m.config.GetLongitude(),
				ObserverElev: m.config.GetElevation(),
				LocationName: m.config.LocationName,
			}
			if err := logStore.CreateSession(sess); err != nil {
				log.Printf(
					"warning: could not create observation session: %v",
					err,
				)
			} else {
				activeSessionID = sess.ID
				unsubAutoLog = logging.RegisterAutoLog(
					slewSvc, logStore, activeSessionID,
				)
			}
		}
	}

	var skyChartPanel Panel
	if cat != nil {
		skyChartPanel = NewSkyChartPanel(cat, m.config, astroSvc, statusPanel, slewSvc)
	} else {
		skyChartPanel = NewPlaceholderPanel("Sky Chart", theme.ColorChromaticIcon())
	}

	var objectsPanel Panel
	if cat != nil {
		obp := NewObjectBrowserPanel(
			cat.Registry, engine, astroSvc, m.config, slewSvc, m.window,
		)
		obp.OnShowOnChart = func(ra, dec float64, catalogID string) {
			m.nav.Select(4) // Sky Chart panel index (unchanged)
			if sp, ok := skyChartPanel.(*SkyChartPanel); ok {
				sp.Chart().CenterOn(ra, dec)
				sp.Chart().HighlightObject(catalogID)
			}
		}
		objectsPanel = obp
	} else {
		objectsPanel = NewPlaceholderPanel("Objects", theme.SearchIcon())
	}

	// Build the log panel (nil store is handled gracefully by a placeholder).
	var logPanel Panel
	if logStore != nil {
		var reg *catalog.CatalogRegistry
		if cat != nil {
			reg = cat.Registry
		}
		logPanel = NewLogPanel(logStore, activeSessionID, reg, m.window)
	} else {
		logPanel = NewPlaceholderPanel("Log", theme.ListIcon())
	}

	// Wire Stellarium server into the settings panel.
	settingsPanel.SetStellariumServer(stelSrv, m.config)

	// Auto-start if configured.
	if stelEnabled {
		if err := stelSrv.Start(); err != nil {
			log.Printf("warning: stellarium server auto-start failed: %v", err)
		}
	}

	// panels[0..5] = status, goto, tracking, alignment, sky chart, objects
	// panels[6]    = log  (new)
	// panels[7]    = settings
	m.panels = []Panel{
		statusPanel,
		gotoPanel,
		trackingPanel,
		NewAlignmentPanel(statusPanel, slewSvc, astroSvc, m.config),
		skyChartPanel,
		objectsPanel,
		logPanel,
		settingsPanel,
	}

	m.content = container.NewStack(m.panels[0].Content())

	m.nav = widget.NewList(
		func() int { return len(m.panels) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.DocumentIcon()),
				widget.NewLabel("Template"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			icon := m.panels[id].Icon()
			if icon == nil {
				icon = theme.SettingsIcon()
			}
			c.Objects[0].(*widget.Icon).SetResource(icon)
			c.Objects[1].(*widget.Label).SetText(m.panels[id].Title())
		},
	)
	m.nav.OnSelected = func(id widget.ListItemID) {
		m.content.Objects = []fyne.CanvasObject{m.panels[id].Content()}
		m.content.Refresh()
	}

	// Select Status panel by default.
	m.nav.Select(0)

	split := container.NewHSplit(m.nav, m.content)
	split.SetOffset(0.2)

	m.window.SetContent(split)
	m.window.Resize(fyne.NewSize(1024, 768))

	// Session lifecycle: close session (and prune if empty) when the window
	// is closed.
	m.window.SetCloseIntercept(func() {
		stelSrv.Stop()

		if unsubAutoLog != nil {
			unsubAutoLog()
		}
		if logStore != nil && activeSessionID != 0 {
			if err := logStore.CloseSession(
				activeSessionID, time.Now().UTC(),
			); err != nil {
				log.Printf("warning: close session: %v", err)
			}
			if err := logStore.DeleteEmptySession(activeSessionID); err != nil {
				log.Printf("warning: delete empty session: %v", err)
			}
			if err := logStore.Close(); err != nil {
				log.Printf("warning: close log store: %v", err)
			}
		}
		m.window.Close()
	})
}
