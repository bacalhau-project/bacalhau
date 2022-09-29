/*
 * IPFS Pinning Service API
 *
 *   ## About this spec The IPFS Pinning Service API is intended to be an implementation-agnostic API&#x3a; - For use and implementation by pinning service providers - For use in client mode by IPFS nodes and GUI-based applications  > **Note**: while ready for implementation, this spec is still a work in progress! 🏗️  **Your input and feedback are welcome and valuable as we develop this API spec. Please join the design discussion at [github.com/ipfs/pinning-services-api-spec](https://github.com/ipfs/pinning-services-api-spec).**  # Schemas This section describes the most important object types and conventions.  A full list of fields and schemas can be found in the `schemas` section of the [YAML file](https://github.com/ipfs/pinning-services-api-spec/blob/master/ipfs-pinning-service.yaml).  ## Identifiers ### cid [Content Identifier (CID)](https://docs.ipfs.io/concepts/content-addressing/) points at the root of a DAG that is pinned recursively. ### requestid Unique identifier of a pin request.  When a pin is created, the service responds with unique `requestid` that can be later used for pin removal. When the same `cid` is pinned again, a different `requestid` is returned to differentiate between those pin requests.  Service implementation should use UUID, `hash(accessToken,Pin,PinStatus.created)`, or any other opaque identifier that provides equally strong protection against race conditions.  ## Objects ### Pin object  ![pin object](https://bafybeideck2fchyxna4wqwc2mo67yriokehw3yujboc5redjdaajrk2fjq.ipfs.dweb.link/pin.png)  The `Pin` object is a representation of a pin request.  It includes the `cid` of data to be pinned, as well as optional metadata in `name`, `origins`, and `meta`.  ### Pin status response  ![pin status response object](https://bafybeideck2fchyxna4wqwc2mo67yriokehw3yujboc5redjdaajrk2fjq.ipfs.dweb.link/pinstatus.png)  The `PinStatus` object is a representation of the current state of a pinning operation. It includes the original `pin` object, along with the current `status` and globally unique `requestid` of the entire pinning request, which can be used for future status checks and management. Addresses in the `delegates` array are peers delegated by the pinning service for facilitating direct file transfers (more details in the provider hints section). Any additional vendor-specific information is returned in optional `info`.  ## The pin lifecycle  ![pinning service objects and lifecycle](https://bafybeideck2fchyxna4wqwc2mo67yriokehw3yujboc5redjdaajrk2fjq.ipfs.dweb.link/lifecycle.png)  ### Creating a new pin object The user sends a `Pin` object to `POST /pins` and receives a `PinStatus` response: - `requestid` in `PinStatus` is the identifier of the pin operation, which can can be used for checking status, and removing the pin in the future - `status` in `PinStatus` indicates the current state of a pin  ### Checking status of in-progress pinning `status` (in `PinStatus`) may indicate a pending state (`queued` or `pinning`). This means the data behind `Pin.cid` was not found on the pinning service and is being fetched from the IPFS network at large, which may take time.  In this case, the user can periodically check pinning progress via `GET /pins/{requestid}` until pinning is successful, or the user decides to remove the pending pin.  ### Replacing an existing pin object The user can replace an existing pin object via `POST /pins/{requestid}`. This is a shortcut for removing a pin object identified by `requestid` and creating a new one in a single API call that protects against undesired garbage collection of blocks common to both pins. Useful when updating a pin representing a huge dataset where most of blocks did not change. The new pin object `requestid` is returned in the `PinStatus` response. The old pin object is deleted automatically.  ### Removing a pin object A pin object can be removed via `DELETE /pins/{requestid}`.   ## Provider hints Pinning of new data can be accelerated by providing a list of known data sources in `Pin.origins`, and connecting at least one of them to pinning service nodes at `PinStatus.delegates`.  The most common scenario is a client putting its own IPFS node's multiaddrs in `Pin.origins`,  and then directly connecting to every multiaddr returned by a pinning service in `PinStatus.delegates` to initiate transfer.  This ensures data transfer starts immediately (without waiting for provider discovery over DHT), and direct dial from a client works around peer routing issues in restrictive network topologies such as NATs.  ## Custom metadata Pinning services are encouraged to add support for additional features by leveraging the optional `Pin.meta` and `PinStatus.info` fields. While these attributes can be application- or vendor-specific, we encourage the community at large to leverage these attributes as a sandbox to come up with conventions that could become part of future revisions of this API. ### Pin metadata String keys and values passed in `Pin.meta` are persisted with the pin object.  Potential uses: - `Pin.meta[app_id]`: Attaching a unique identifier to pins created by an app enables filtering pins per app via `?meta={\"app_id\":<UUID>}` - `Pin.meta[vendor_policy]`: Vendor-specific policy (for example: which region to use, how many copies to keep)  Note that it is OK for a client to omit or ignore these optional attributes; doing so should not impact the basic pinning functionality.  ### Pin status info Additional `PinStatus.info` can be returned by pinning service.  Potential uses: - `PinStatus.info[status_details]`: more info about the current status (queue position, percentage of transferred data, summary of where data is stored, etc); when `PinStatus.status=failed`, it could provide a reason why a pin operation failed (e.g. lack of funds, DAG too big, etc.) - `PinStatus.info[dag_size]`: the size of pinned data, along with DAG overhead - `PinStatus.info[raw_size]`: the size of data without DAG overhead (eg. unixfs) - `PinStatus.info[pinned_until]`: if vendor supports time-bound pins, this could indicate when the pin will expire  # Pagination and filtering Pin objects can be listed by executing `GET /pins` with optional parameters:  - When no filters are provided, the endpoint will return a small batch of the 10 most recently created items, from the latest to the oldest. - The number of returned items can be adjusted with the `limit` parameter (implicit default is 10). - If the value in `PinResults.count` is bigger than the length of `PinResults.results`, the client can infer there are more results that can be queried. - To read more items, pass the `before` filter with the timestamp from `PinStatus.created` found in the oldest item in the current batch of results. Repeat to read all results. - Returned results can be fine-tuned by applying optional `after`, `cid`, `name`, `status`, or `meta` filters.  > **Note**: pagination by the `created` timestamp requires each value to be globally unique. Any future considerations to add support for bulk creation must account for this.
 *
 * API version: 0.1.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"encoding/json"
	"time"
)

// PinStatus Pin object with status
type PinStatus struct {
	// Globally unique identifier of the pin request; can be used to check the status of ongoing pinning, or pin removal
	Requestid string `json:"requestid"`
	Status    Status `json:"status"`
	// Immutable timestamp indicating when a pin request entered a pinning service; can be used for filtering results and pagination
	Created time.Time `json:"created"`
	Pin     Pin       `json:"pin"`
	// List of multiaddrs designated by pinning service for transferring any new data from external peers
	Delegates []string `json:"delegates"`
	// Optional info for PinStatus response
	Info *map[string]string `json:"info,omitempty"`
}

// NewPinStatus instantiates a new PinStatus object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewPinStatus(requestid string, status Status, created time.Time, pin Pin, delegates []string) *PinStatus {
	this := PinStatus{}
	this.Requestid = requestid
	this.Status = status
	this.Created = created
	this.Pin = pin
	this.Delegates = delegates
	return &this
}

// NewPinStatusWithDefaults instantiates a new PinStatus object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewPinStatusWithDefaults() *PinStatus {
	this := PinStatus{}
	return &this
}

// GetRequestid returns the Requestid field value
func (o *PinStatus) GetRequestid() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Requestid
}

// GetRequestidOk returns a tuple with the Requestid field value
// and a boolean to check if the value has been set.
func (o *PinStatus) GetRequestidOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Requestid, true
}

// SetRequestid sets field value
func (o *PinStatus) SetRequestid(v string) {
	o.Requestid = v
}

// GetStatus returns the Status field value
func (o *PinStatus) GetStatus() Status {
	if o == nil {
		var ret Status
		return ret
	}

	return o.Status
}

// GetStatusOk returns a tuple with the Status field value
// and a boolean to check if the value has been set.
func (o *PinStatus) GetStatusOk() (*Status, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Status, true
}

// SetStatus sets field value
func (o *PinStatus) SetStatus(v Status) {
	o.Status = v
}

// GetCreated returns the Created field value
func (o *PinStatus) GetCreated() time.Time {
	if o == nil {
		var ret time.Time
		return ret
	}

	return o.Created
}

// GetCreatedOk returns a tuple with the Created field value
// and a boolean to check if the value has been set.
func (o *PinStatus) GetCreatedOk() (*time.Time, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Created, true
}

// SetCreated sets field value
func (o *PinStatus) SetCreated(v time.Time) {
	o.Created = v
}

// GetPin returns the Pin field value
func (o *PinStatus) GetPin() Pin {
	if o == nil {
		var ret Pin
		return ret
	}

	return o.Pin
}

// GetPinOk returns a tuple with the Pin field value
// and a boolean to check if the value has been set.
func (o *PinStatus) GetPinOk() (*Pin, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Pin, true
}

// SetPin sets field value
func (o *PinStatus) SetPin(v Pin) {
	o.Pin = v
}

// GetDelegates returns the Delegates field value
func (o *PinStatus) GetDelegates() []string {
	if o == nil {
		var ret []string
		return ret
	}

	return o.Delegates
}

// GetDelegatesOk returns a tuple with the Delegates field value
// and a boolean to check if the value has been set.
func (o *PinStatus) GetDelegatesOk() (*[]string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Delegates, true
}

// SetDelegates sets field value
func (o *PinStatus) SetDelegates(v []string) {
	o.Delegates = v
}

// GetInfo returns the Info field value if set, zero value otherwise.
func (o *PinStatus) GetInfo() map[string]string {
	if o == nil || o.Info == nil {
		var ret map[string]string
		return ret
	}
	return *o.Info
}

// GetInfoOk returns a tuple with the Info field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *PinStatus) GetInfoOk() (*map[string]string, bool) {
	if o == nil || o.Info == nil {
		return nil, false
	}
	return o.Info, true
}

// HasInfo returns a boolean if a field has been set.
func (o *PinStatus) HasInfo() bool {
	if o != nil && o.Info != nil {
		return true
	}

	return false
}

// SetInfo gets a reference to the given map[string]string and assigns it to the Info field.
func (o *PinStatus) SetInfo(v map[string]string) {
	o.Info = &v
}

func (o PinStatus) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["requestid"] = o.Requestid
	}
	if true {
		toSerialize["status"] = o.Status
	}
	if true {
		toSerialize["created"] = o.Created
	}
	if true {
		toSerialize["pin"] = o.Pin
	}
	if true {
		toSerialize["delegates"] = o.Delegates
	}
	if o.Info != nil {
		toSerialize["info"] = o.Info
	}
	return json.Marshal(toSerialize)
}

type NullablePinStatus struct {
	value *PinStatus
	isSet bool
}

func (v NullablePinStatus) Get() *PinStatus {
	return v.value
}

func (v *NullablePinStatus) Set(val *PinStatus) {
	v.value = val
	v.isSet = true
}

func (v NullablePinStatus) IsSet() bool {
	return v.isSet
}

func (v *NullablePinStatus) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullablePinStatus(val *PinStatus) *NullablePinStatus {
	return &NullablePinStatus{value: val, isSet: true}
}

func (v NullablePinStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullablePinStatus) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}
