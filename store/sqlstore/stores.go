package sqlstore

type DeviceStore struct {
	*NeonizeSQLStore
}

type IdentityStore struct {
	*NeonizeSQLStore
}

type PreKeyStore struct {
	*NeonizeSQLStore
}

type SessionStore struct {
	*NeonizeSQLStore
}

type ContactStore struct {
	*NeonizeSQLStore
}

type ChatSettingsStore struct {
	*NeonizeSQLStore
}

type MessageSecretStore struct {
	*NeonizeSQLStore
}

type PrivacyTokenStore struct {
	*NeonizeSQLStore
}

type LIDMapStore struct {
	*NeonizeSQLStore
}

type EventBufferStore struct {
	*NeonizeSQLStore
}
