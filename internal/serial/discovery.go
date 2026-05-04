package serial

import (
	"runtime"
	"strings"

	hwserial "go.bug.st/serial"
)

// ListPorts returns a filtered list of available serial port names.
// On macOS, only /dev/cu.* ports are included (outgoing connections).
// On Linux, only /dev/ttyUSB* and /dev/ttyACM* ports are included.
// On Windows, all COM* ports are included.
func ListPorts() ([]string, error) {
	ports, err := hwserial.GetPortsList()
	if err != nil {
		return nil, err
	}

	var filtered []string
	for _, p := range ports {
		switch runtime.GOOS {
		case "darwin":
			if strings.HasPrefix(p, "/dev/cu.") && !strings.Contains(p, "Bluetooth") && !strings.Contains(p, "debug") {
				filtered = append(filtered, p)
			}
		case "linux":
			if strings.HasPrefix(p, "/dev/ttyUSB") || strings.HasPrefix(p, "/dev/ttyACM") {
				filtered = append(filtered, p)
			}
		default:
			// Windows: include all (COM ports)
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}
