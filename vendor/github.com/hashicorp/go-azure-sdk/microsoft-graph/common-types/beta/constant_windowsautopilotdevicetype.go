package beta

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Copyright (c) HashiCorp Inc. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type WindowsAutopilotDeviceType string

const (
	WindowsAutopilotDeviceType_HoloLens       WindowsAutopilotDeviceType = "holoLens"
	WindowsAutopilotDeviceType_SurfaceHub2    WindowsAutopilotDeviceType = "surfaceHub2"
	WindowsAutopilotDeviceType_SurfaceHub2S   WindowsAutopilotDeviceType = "surfaceHub2S"
	WindowsAutopilotDeviceType_VirtualMachine WindowsAutopilotDeviceType = "virtualMachine"
	WindowsAutopilotDeviceType_WindowsPc      WindowsAutopilotDeviceType = "windowsPc"
)

func PossibleValuesForWindowsAutopilotDeviceType() []string {
	return []string{
		string(WindowsAutopilotDeviceType_HoloLens),
		string(WindowsAutopilotDeviceType_SurfaceHub2),
		string(WindowsAutopilotDeviceType_SurfaceHub2S),
		string(WindowsAutopilotDeviceType_VirtualMachine),
		string(WindowsAutopilotDeviceType_WindowsPc),
	}
}

func (s *WindowsAutopilotDeviceType) UnmarshalJSON(bytes []byte) error {
	var decoded string
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		return fmt.Errorf("unmarshaling: %+v", err)
	}
	out, err := parseWindowsAutopilotDeviceType(decoded)
	if err != nil {
		return fmt.Errorf("parsing %q: %+v", decoded, err)
	}
	*s = *out
	return nil
}

func parseWindowsAutopilotDeviceType(input string) (*WindowsAutopilotDeviceType, error) {
	vals := map[string]WindowsAutopilotDeviceType{
		"hololens":       WindowsAutopilotDeviceType_HoloLens,
		"surfacehub2":    WindowsAutopilotDeviceType_SurfaceHub2,
		"surfacehub2s":   WindowsAutopilotDeviceType_SurfaceHub2S,
		"virtualmachine": WindowsAutopilotDeviceType_VirtualMachine,
		"windowspc":      WindowsAutopilotDeviceType_WindowsPc,
	}
	if v, ok := vals[strings.ToLower(input)]; ok {
		return &v, nil
	}

	// otherwise presume it's an undefined value and best-effort it
	out := WindowsAutopilotDeviceType(input)
	return &out, nil
}