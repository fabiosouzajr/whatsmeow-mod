package sqlstore

import (
	"database/sql"
)

type NeonizeSQLStore struct {
	*Container
	db *sql.DB

	Device         *DeviceStore
	Identities     *IdentityStore
	PreKeys        *PreKeyStore
	Sessions       *SessionStore
	Contacts       *ContactStore
	ChatSettings   *ChatSettingsStore
	MessageSecrets *MessageSecretStore
	PrivacyTokens  *PrivacyTokenStore
	LIDMap         *LIDMapStore
	EventBuffer    *EventBufferStore
	Neonize        *NeonizeStore
}

func NewNeonizeSQLStore(db *sql.DB, container *Container) *NeonizeSQLStore {
	store := &NeonizeSQLStore{
		Container: container,
		db:        db,
	}
	store.Device = &DeviceStore{store}
	store.Identities = &IdentityStore{store}
	store.PreKeys = &PreKeyStore{store}
	store.Sessions = &SessionStore{store}
	store.Contacts = &ContactStore{store}
	store.ChatSettings = &ChatSettingsStore{store}
	store.MessageSecrets = &MessageSecretStore{store}
	store.PrivacyTokens = &PrivacyTokenStore{store}
	store.LIDMap = &LIDMapStore{store}
	store.EventBuffer = &EventBufferStore{store}
	store.Neonize = &NeonizeStore{store}
	return store
}
