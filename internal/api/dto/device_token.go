package dto

// RegisterDeviceTokenRequest represents a request to register a device token
type RegisterDeviceTokenRequest struct {
	ExpoPushToken string `json:"expoPushToken" validate:"required"`
	DeviceName    string `json:"deviceName"`
	Platform      string `json:"platform" validate:"required,oneof=ios android"`
}

// UnregisterDeviceTokenRequest represents a request to unregister a device token
type UnregisterDeviceTokenRequest struct {
	ExpoPushToken string `json:"expoPushToken" validate:"required"`
}
