package sqlstore

import (
	"encoding/json"

	"go.mau.fi/whatsmeow/types"
)

// StoreMessage stores a message in the database
func (s *NeonizeSQLStore) StoreMessage(uuid []byte, messageBytes []byte) error {
	var msg Message
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return err
	}
	return s.Neonize.StoreMessage(uuid, messageBytes)
}

// GetMessage retrieves a message from the database
func (s *NeonizeSQLStore) GetMessage(uuid []byte, chatJIDStr string, messageID string) ([]byte, error) {
	chatJID, err := types.ParseJID(chatJIDStr)
	if err != nil {
		return nil, err
	}

	msg, err := s.Neonize.GetMessage(uuid, chatJID, messageID)
	if err != nil {
		return nil, err
	}

	return json.Marshal(msg)
}

// StoreMedia stores media information in the database
func (s *NeonizeSQLStore) StoreMedia(uuid []byte, mediaBytes []byte) error {
	var media Media
	if err := json.Unmarshal(mediaBytes, &media); err != nil {
		return err
	}
	return s.Neonize.StoreMedia(uuid, mediaBytes)
}

// GetMedia retrieves media information from the database
func (s *NeonizeSQLStore) GetMedia(uuid []byte, mediaID string) ([]byte, error) {
	media, err := s.Neonize.GetMedia(uuid, mediaID)
	if err != nil {
		return nil, err
	}

	return json.Marshal(media)
}

// StoreGroupMessage stores a group message in the database
func (s *NeonizeSQLStore) StoreGroupMessage(uuid []byte, messageBytes []byte) error {
	var msg GroupMessage
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return err
	}
	return s.Neonize.StoreGroupMessage(uuid, messageBytes)
}

// StoreGroupParticipant stores a group participant in the database
func (s *NeonizeSQLStore) StoreGroupParticipant(uuid []byte, participantBytes []byte) error {
	var participant GroupParticipant
	if err := json.Unmarshal(participantBytes, &participant); err != nil {
		return err
	}
	return s.Neonize.StoreGroupParticipant(uuid, participantBytes)
}

// StoreGroupInviteLink stores a group invite link in the database
func (s *NeonizeSQLStore) StoreGroupInviteLink(uuid []byte, linkBytes []byte) error {
	var link GroupInviteLink
	if err := json.Unmarshal(linkBytes, &link); err != nil {
		return err
	}
	return s.Neonize.StoreGroupInviteLink(uuid, linkBytes)
}

// StoreNewsletterSubscription stores a newsletter subscription in the database
func (s *NeonizeSQLStore) StoreNewsletterSubscription(uuid []byte, subBytes []byte) error {
	var sub NewsletterSubscription
	if err := json.Unmarshal(subBytes, &sub); err != nil {
		return err
	}
	return s.Neonize.StoreNewsletterSubscription(uuid, subBytes)
}

// StoreNewsletterMessage stores a newsletter message in the database
func (s *NeonizeSQLStore) StoreNewsletterMessage(uuid []byte, messageBytes []byte) error {
	var msg NewsletterMessage
	if err := json.Unmarshal(messageBytes, &msg); err != nil {
		return err
	}
	return s.Neonize.StoreNewsletterMessage(uuid, messageBytes)
}

// StoreCall stores a call in the database
func (s *NeonizeSQLStore) StoreCall(uuid []byte, callBytes []byte) error {
	var call Call
	if err := json.Unmarshal(callBytes, &call); err != nil {
		return err
	}
	return s.Neonize.StoreCall(uuid, callBytes)
}
