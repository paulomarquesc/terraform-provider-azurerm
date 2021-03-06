package healthcareapis

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

// Kind enumerates the values for kind.
type Kind string

const (
	// Fhir ...
	Fhir Kind = "fhir"
	// FhirR4 ...
	FhirR4 Kind = "fhir-R4"
	// FhirStu3 ...
	FhirStu3 Kind = "fhir-Stu3"
)

// PossibleKindValues returns an array of possible values for the Kind const type.
func PossibleKindValues() []Kind {
	return []Kind{Fhir, FhirR4, FhirStu3}
}

// ManagedServiceIdentityType enumerates the values for managed service identity type.
type ManagedServiceIdentityType string

const (
	// None ...
	None ManagedServiceIdentityType = "None"
	// SystemAssigned ...
	SystemAssigned ManagedServiceIdentityType = "SystemAssigned"
)

// PossibleManagedServiceIdentityTypeValues returns an array of possible values for the ManagedServiceIdentityType const type.
func PossibleManagedServiceIdentityTypeValues() []ManagedServiceIdentityType {
	return []ManagedServiceIdentityType{None, SystemAssigned}
}

// OperationResultStatus enumerates the values for operation result status.
type OperationResultStatus string

const (
	// Canceled ...
	Canceled OperationResultStatus = "Canceled"
	// Failed ...
	Failed OperationResultStatus = "Failed"
	// Requested ...
	Requested OperationResultStatus = "Requested"
	// Running ...
	Running OperationResultStatus = "Running"
	// Succeeded ...
	Succeeded OperationResultStatus = "Succeeded"
)

// PossibleOperationResultStatusValues returns an array of possible values for the OperationResultStatus const type.
func PossibleOperationResultStatusValues() []OperationResultStatus {
	return []OperationResultStatus{Canceled, Failed, Requested, Running, Succeeded}
}

// PrivateEndpointConnectionProvisioningState enumerates the values for private endpoint connection
// provisioning state.
type PrivateEndpointConnectionProvisioningState string

const (
	// PrivateEndpointConnectionProvisioningStateCreating ...
	PrivateEndpointConnectionProvisioningStateCreating PrivateEndpointConnectionProvisioningState = "Creating"
	// PrivateEndpointConnectionProvisioningStateDeleting ...
	PrivateEndpointConnectionProvisioningStateDeleting PrivateEndpointConnectionProvisioningState = "Deleting"
	// PrivateEndpointConnectionProvisioningStateFailed ...
	PrivateEndpointConnectionProvisioningStateFailed PrivateEndpointConnectionProvisioningState = "Failed"
	// PrivateEndpointConnectionProvisioningStateSucceeded ...
	PrivateEndpointConnectionProvisioningStateSucceeded PrivateEndpointConnectionProvisioningState = "Succeeded"
)

// PossiblePrivateEndpointConnectionProvisioningStateValues returns an array of possible values for the PrivateEndpointConnectionProvisioningState const type.
func PossiblePrivateEndpointConnectionProvisioningStateValues() []PrivateEndpointConnectionProvisioningState {
	return []PrivateEndpointConnectionProvisioningState{PrivateEndpointConnectionProvisioningStateCreating, PrivateEndpointConnectionProvisioningStateDeleting, PrivateEndpointConnectionProvisioningStateFailed, PrivateEndpointConnectionProvisioningStateSucceeded}
}

// PrivateEndpointServiceConnectionStatus enumerates the values for private endpoint service connection status.
type PrivateEndpointServiceConnectionStatus string

const (
	// Approved ...
	Approved PrivateEndpointServiceConnectionStatus = "Approved"
	// Pending ...
	Pending PrivateEndpointServiceConnectionStatus = "Pending"
	// Rejected ...
	Rejected PrivateEndpointServiceConnectionStatus = "Rejected"
)

// PossiblePrivateEndpointServiceConnectionStatusValues returns an array of possible values for the PrivateEndpointServiceConnectionStatus const type.
func PossiblePrivateEndpointServiceConnectionStatusValues() []PrivateEndpointServiceConnectionStatus {
	return []PrivateEndpointServiceConnectionStatus{Approved, Pending, Rejected}
}

// ProvisioningState enumerates the values for provisioning state.
type ProvisioningState string

const (
	// ProvisioningStateAccepted ...
	ProvisioningStateAccepted ProvisioningState = "Accepted"
	// ProvisioningStateCanceled ...
	ProvisioningStateCanceled ProvisioningState = "Canceled"
	// ProvisioningStateCreating ...
	ProvisioningStateCreating ProvisioningState = "Creating"
	// ProvisioningStateDeleting ...
	ProvisioningStateDeleting ProvisioningState = "Deleting"
	// ProvisioningStateDeprovisioned ...
	ProvisioningStateDeprovisioned ProvisioningState = "Deprovisioned"
	// ProvisioningStateFailed ...
	ProvisioningStateFailed ProvisioningState = "Failed"
	// ProvisioningStateSucceeded ...
	ProvisioningStateSucceeded ProvisioningState = "Succeeded"
	// ProvisioningStateUpdating ...
	ProvisioningStateUpdating ProvisioningState = "Updating"
	// ProvisioningStateVerifying ...
	ProvisioningStateVerifying ProvisioningState = "Verifying"
)

// PossibleProvisioningStateValues returns an array of possible values for the ProvisioningState const type.
func PossibleProvisioningStateValues() []ProvisioningState {
	return []ProvisioningState{ProvisioningStateAccepted, ProvisioningStateCanceled, ProvisioningStateCreating, ProvisioningStateDeleting, ProvisioningStateDeprovisioned, ProvisioningStateFailed, ProvisioningStateSucceeded, ProvisioningStateUpdating, ProvisioningStateVerifying}
}

// PublicNetworkAccess enumerates the values for public network access.
type PublicNetworkAccess string

const (
	// Disabled ...
	Disabled PublicNetworkAccess = "Disabled"
	// Enabled ...
	Enabled PublicNetworkAccess = "Enabled"
)

// PossiblePublicNetworkAccessValues returns an array of possible values for the PublicNetworkAccess const type.
func PossiblePublicNetworkAccessValues() []PublicNetworkAccess {
	return []PublicNetworkAccess{Disabled, Enabled}
}

// ServiceNameUnavailabilityReason enumerates the values for service name unavailability reason.
type ServiceNameUnavailabilityReason string

const (
	// AlreadyExists ...
	AlreadyExists ServiceNameUnavailabilityReason = "AlreadyExists"
	// Invalid ...
	Invalid ServiceNameUnavailabilityReason = "Invalid"
)

// PossibleServiceNameUnavailabilityReasonValues returns an array of possible values for the ServiceNameUnavailabilityReason const type.
func PossibleServiceNameUnavailabilityReasonValues() []ServiceNameUnavailabilityReason {
	return []ServiceNameUnavailabilityReason{AlreadyExists, Invalid}
}
