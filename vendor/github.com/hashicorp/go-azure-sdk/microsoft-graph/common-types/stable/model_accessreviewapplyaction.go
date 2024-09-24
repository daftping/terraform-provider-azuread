package stable

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Copyright (c) HashiCorp Inc. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type AccessReviewApplyAction interface {
	AccessReviewApplyAction() BaseAccessReviewApplyActionImpl
}

var _ AccessReviewApplyAction = BaseAccessReviewApplyActionImpl{}

type BaseAccessReviewApplyActionImpl struct {
	// The OData ID of this entity
	ODataId *string `json:"@odata.id,omitempty"`

	// The OData Type of this entity
	ODataType *string `json:"@odata.type,omitempty"`

	// Model Behaviors
	OmitDiscriminatedValue bool `json:"-"`
}

func (s BaseAccessReviewApplyActionImpl) AccessReviewApplyAction() BaseAccessReviewApplyActionImpl {
	return s
}

var _ AccessReviewApplyAction = RawAccessReviewApplyActionImpl{}

// RawAccessReviewApplyActionImpl is returned when the Discriminated Value doesn't match any of the defined types
// NOTE: this should only be used when a type isn't defined for this type of Object (as a workaround)
// and is used only for Deserialization (e.g. this cannot be used as a Request Payload).
type RawAccessReviewApplyActionImpl struct {
	accessReviewApplyAction BaseAccessReviewApplyActionImpl
	Type                    string
	Values                  map[string]interface{}
}

func (s RawAccessReviewApplyActionImpl) AccessReviewApplyAction() BaseAccessReviewApplyActionImpl {
	return s.accessReviewApplyAction
}

func UnmarshalAccessReviewApplyActionImplementation(input []byte) (AccessReviewApplyAction, error) {
	if input == nil {
		return nil, nil
	}

	var temp map[string]interface{}
	if err := json.Unmarshal(input, &temp); err != nil {
		return nil, fmt.Errorf("unmarshaling AccessReviewApplyAction into map[string]interface: %+v", err)
	}

	var value string
	if v, ok := temp["@odata.type"]; ok {
		value = fmt.Sprintf("%v", v)
	}

	if strings.EqualFold(value, "#microsoft.graph.disableAndDeleteUserApplyAction") {
		var out DisableAndDeleteUserApplyAction
		if err := json.Unmarshal(input, &out); err != nil {
			return nil, fmt.Errorf("unmarshaling into DisableAndDeleteUserApplyAction: %+v", err)
		}
		return out, nil
	}

	if strings.EqualFold(value, "#microsoft.graph.removeAccessApplyAction") {
		var out RemoveAccessApplyAction
		if err := json.Unmarshal(input, &out); err != nil {
			return nil, fmt.Errorf("unmarshaling into RemoveAccessApplyAction: %+v", err)
		}
		return out, nil
	}

	var parent BaseAccessReviewApplyActionImpl
	if err := json.Unmarshal(input, &parent); err != nil {
		return nil, fmt.Errorf("unmarshaling into BaseAccessReviewApplyActionImpl: %+v", err)
	}

	return RawAccessReviewApplyActionImpl{
		accessReviewApplyAction: parent,
		Type:                    value,
		Values:                  temp,
	}, nil

}