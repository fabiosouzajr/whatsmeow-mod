package main

import (
	"C"
	"encoding/json"
	"unsafe"

	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
)

//export StoreMessage
func StoreMessage(storePtr unsafe.Pointer, uuid *C.char, messageBytes *C.char, messageLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreMessage([]byte(C.GoString(uuid)), []byte(C.GoStringN(messageBytes, messageLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export GetMessage
func GetMessage(storePtr unsafe.Pointer, uuid *C.char, chatJIDStr *C.char, messageID *C.char, result **C.char) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	userJID, err := types.ParseJID(C.GoString(uuid))
	if err != nil {
		return C.int(1)
	}
	chatJID, err := types.ParseJID(C.GoString(chatJIDStr))
	if err != nil {
		return C.int(1)
	}
	msg, err := store.Neonize.GetMessage([]byte(userJID.String()), chatJID, C.GoString(messageID))
	if err != nil {
		return C.int(1)
	}
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return C.int(1)
	}
	*result = C.CString(string(msgJSON))
	return C.int(0)
}

//export StoreMedia
func StoreMedia(storePtr unsafe.Pointer, uuid *C.char, mediaBytes *C.char, mediaLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreMedia([]byte(C.GoString(uuid)), []byte(C.GoStringN(mediaBytes, mediaLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export GetMedia
func GetMedia(storePtr unsafe.Pointer, uuid *C.char, mediaID *C.char, result **C.char) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	userJID, err := types.ParseJID(C.GoString(uuid))
	if err != nil {
		return C.int(1)
	}
	media, err := store.Neonize.GetMedia([]byte(userJID.String()), C.GoString(mediaID))
	if err != nil {
		return C.int(1)
	}
	mediaJSON, err := json.Marshal(media)
	if err != nil {
		return C.int(1)
	}
	*result = C.CString(string(mediaJSON))
	return C.int(0)
}

//export StoreGroupMessage
func StoreGroupMessage(storePtr unsafe.Pointer, uuid *C.char, messageBytes *C.char, messageLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreGroupMessage([]byte(C.GoString(uuid)), []byte(C.GoStringN(messageBytes, messageLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export StoreGroupParticipant
func StoreGroupParticipant(storePtr unsafe.Pointer, uuid *C.char, participantBytes *C.char, participantLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreGroupParticipant([]byte(C.GoString(uuid)), []byte(C.GoStringN(participantBytes, participantLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export StoreGroupInviteLink
func StoreGroupInviteLink(storePtr unsafe.Pointer, uuid *C.char, linkBytes *C.char, linkLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreGroupInviteLink([]byte(C.GoString(uuid)), []byte(C.GoStringN(linkBytes, linkLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export StoreNewsletterSubscription
func StoreNewsletterSubscription(storePtr unsafe.Pointer, uuid *C.char, subBytes *C.char, subLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreNewsletterSubscription([]byte(C.GoString(uuid)), []byte(C.GoStringN(subBytes, subLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export StoreNewsletterMessage
func StoreNewsletterMessage(storePtr unsafe.Pointer, uuid *C.char, messageBytes *C.char, messageLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreNewsletterMessage([]byte(C.GoString(uuid)), []byte(C.GoStringN(messageBytes, messageLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}

//export StoreCall
func StoreCall(storePtr unsafe.Pointer, uuid *C.char, callBytes *C.char, callLen C.int) C.int {
	store := (*sqlstore.NeonizeSQLStore)(storePtr)
	err := store.Neonize.StoreCall([]byte(C.GoString(uuid)), []byte(C.GoStringN(callBytes, callLen)))
	if err != nil {
		return C.int(1)
	}
	return C.int(0)
}
