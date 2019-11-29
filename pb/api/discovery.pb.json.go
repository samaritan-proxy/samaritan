// Code generated by protoc-gen-jsonify. DO NOT EDIT.
// source: api/discovery.proto

package api
import (
	"bytes"
	"encoding/json"
	"github.com/gogo/protobuf/jsonpb"
)

// DependencyDiscoveryRequestJSONMarshaler describes the default jsonpb.Marshaler used by all
// instances of DependencyDiscoveryRequest. This struct is safe to replace or modify but
// should not be done so concurrently.
var DependencyDiscoveryRequestJSONMarshaler = new(jsonpb.Marshaler)
// MarshalJSON satisfies the encoding/json Marshaler interface. This method
// uses the more correct jsonpb package to correctly marshal the message.
func (m *DependencyDiscoveryRequest) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	buf := &bytes.Buffer{}
	if err := DependencyDiscoveryRequestJSONMarshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}
	return buf.Bytes(), nil
}
var _ json.Marshaler = (*DependencyDiscoveryRequest)(nil)
// DependencyDiscoveryRequestJSONUnmarshaler describes the default jsonpb.Unmarshaler used by all
// instances of DependencyDiscoveryRequest. This struct is safe to replace or modify but
// should not be done so concurrently.
var DependencyDiscoveryRequestJSONUnmarshaler = new(jsonpb.Unmarshaler)
// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *DependencyDiscoveryRequest) UnmarshalJSON(b []byte) error {
	return DependencyDiscoveryRequestJSONUnmarshaler.Unmarshal(bytes.NewReader(b), m)
}
var _ json.Unmarshaler = (*DependencyDiscoveryRequest)(nil)

// DependencyDiscoveryResponseJSONMarshaler describes the default jsonpb.Marshaler used by all
// instances of DependencyDiscoveryResponse. This struct is safe to replace or modify but
// should not be done so concurrently.
var DependencyDiscoveryResponseJSONMarshaler = new(jsonpb.Marshaler)
// MarshalJSON satisfies the encoding/json Marshaler interface. This method
// uses the more correct jsonpb package to correctly marshal the message.
func (m *DependencyDiscoveryResponse) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	buf := &bytes.Buffer{}
	if err := DependencyDiscoveryResponseJSONMarshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}
	return buf.Bytes(), nil
}
var _ json.Marshaler = (*DependencyDiscoveryResponse)(nil)
// DependencyDiscoveryResponseJSONUnmarshaler describes the default jsonpb.Unmarshaler used by all
// instances of DependencyDiscoveryResponse. This struct is safe to replace or modify but
// should not be done so concurrently.
var DependencyDiscoveryResponseJSONUnmarshaler = new(jsonpb.Unmarshaler)
// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *DependencyDiscoveryResponse) UnmarshalJSON(b []byte) error {
	return DependencyDiscoveryResponseJSONUnmarshaler.Unmarshal(bytes.NewReader(b), m)
}
var _ json.Unmarshaler = (*DependencyDiscoveryResponse)(nil)

// SvcConfigDiscoveryRequestJSONMarshaler describes the default jsonpb.Marshaler used by all
// instances of SvcConfigDiscoveryRequest. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcConfigDiscoveryRequestJSONMarshaler = new(jsonpb.Marshaler)
// MarshalJSON satisfies the encoding/json Marshaler interface. This method
// uses the more correct jsonpb package to correctly marshal the message.
func (m *SvcConfigDiscoveryRequest) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	buf := &bytes.Buffer{}
	if err := SvcConfigDiscoveryRequestJSONMarshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}
	return buf.Bytes(), nil
}
var _ json.Marshaler = (*SvcConfigDiscoveryRequest)(nil)
// SvcConfigDiscoveryRequestJSONUnmarshaler describes the default jsonpb.Unmarshaler used by all
// instances of SvcConfigDiscoveryRequest. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcConfigDiscoveryRequestJSONUnmarshaler = new(jsonpb.Unmarshaler)
// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *SvcConfigDiscoveryRequest) UnmarshalJSON(b []byte) error {
	return SvcConfigDiscoveryRequestJSONUnmarshaler.Unmarshal(bytes.NewReader(b), m)
}
var _ json.Unmarshaler = (*SvcConfigDiscoveryRequest)(nil)

// SvcConfigDiscoveryResponseJSONMarshaler describes the default jsonpb.Marshaler used by all
// instances of SvcConfigDiscoveryResponse. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcConfigDiscoveryResponseJSONMarshaler = new(jsonpb.Marshaler)
// MarshalJSON satisfies the encoding/json Marshaler interface. This method
// uses the more correct jsonpb package to correctly marshal the message.
func (m *SvcConfigDiscoveryResponse) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	buf := &bytes.Buffer{}
	if err := SvcConfigDiscoveryResponseJSONMarshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}
	return buf.Bytes(), nil
}
var _ json.Marshaler = (*SvcConfigDiscoveryResponse)(nil)
// SvcConfigDiscoveryResponseJSONUnmarshaler describes the default jsonpb.Unmarshaler used by all
// instances of SvcConfigDiscoveryResponse. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcConfigDiscoveryResponseJSONUnmarshaler = new(jsonpb.Unmarshaler)
// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *SvcConfigDiscoveryResponse) UnmarshalJSON(b []byte) error {
	return SvcConfigDiscoveryResponseJSONUnmarshaler.Unmarshal(bytes.NewReader(b), m)
}
var _ json.Unmarshaler = (*SvcConfigDiscoveryResponse)(nil)

// SvcEndpointDiscoveryRequestJSONMarshaler describes the default jsonpb.Marshaler used by all
// instances of SvcEndpointDiscoveryRequest. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcEndpointDiscoveryRequestJSONMarshaler = new(jsonpb.Marshaler)
// MarshalJSON satisfies the encoding/json Marshaler interface. This method
// uses the more correct jsonpb package to correctly marshal the message.
func (m *SvcEndpointDiscoveryRequest) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	buf := &bytes.Buffer{}
	if err := SvcEndpointDiscoveryRequestJSONMarshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}
	return buf.Bytes(), nil
}
var _ json.Marshaler = (*SvcEndpointDiscoveryRequest)(nil)
// SvcEndpointDiscoveryRequestJSONUnmarshaler describes the default jsonpb.Unmarshaler used by all
// instances of SvcEndpointDiscoveryRequest. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcEndpointDiscoveryRequestJSONUnmarshaler = new(jsonpb.Unmarshaler)
// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *SvcEndpointDiscoveryRequest) UnmarshalJSON(b []byte) error {
	return SvcEndpointDiscoveryRequestJSONUnmarshaler.Unmarshal(bytes.NewReader(b), m)
}
var _ json.Unmarshaler = (*SvcEndpointDiscoveryRequest)(nil)

// SvcEndpointDiscoveryResponseJSONMarshaler describes the default jsonpb.Marshaler used by all
// instances of SvcEndpointDiscoveryResponse. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcEndpointDiscoveryResponseJSONMarshaler = new(jsonpb.Marshaler)
// MarshalJSON satisfies the encoding/json Marshaler interface. This method
// uses the more correct jsonpb package to correctly marshal the message.
func (m *SvcEndpointDiscoveryResponse) MarshalJSON() ([]byte, error) {
	if m == nil {
		return json.Marshal(nil)
	}
	buf := &bytes.Buffer{}
	if err := SvcEndpointDiscoveryResponseJSONMarshaler.Marshal(buf, m); err != nil {
	  return nil, err
	}
	return buf.Bytes(), nil
}
var _ json.Marshaler = (*SvcEndpointDiscoveryResponse)(nil)
// SvcEndpointDiscoveryResponseJSONUnmarshaler describes the default jsonpb.Unmarshaler used by all
// instances of SvcEndpointDiscoveryResponse. This struct is safe to replace or modify but
// should not be done so concurrently.
var SvcEndpointDiscoveryResponseJSONUnmarshaler = new(jsonpb.Unmarshaler)
// UnmarshalJSON satisfies the encoding/json Unmarshaler interface. This method
// uses the more correct jsonpb package to correctly unmarshal the message.
func (m *SvcEndpointDiscoveryResponse) UnmarshalJSON(b []byte) error {
	return SvcEndpointDiscoveryResponseJSONUnmarshaler.Unmarshal(bytes.NewReader(b), m)
}
var _ json.Unmarshaler = (*SvcEndpointDiscoveryResponse)(nil)

