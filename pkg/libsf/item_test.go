package libsf_test

import (
	"strings"
	"testing"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestItem_Seal3(t *testing.T) {
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion3,
		MasterKey: "f07d6183b8ea8c50900cf4a767a4d5aeb6fbe821dd6c514f15ec0df8f74d282b",
		AuthKey:   "45318023da0253ac5f28fe3cf48c5a8345df9f720e5fc082c8ba226b28446026",
	}
	item := &libsf.Item{
		ID:          "d989ccc9-15c6-475e-839b-1690bd07d073",
		UserID:      "b329a187-ddf8-4e9b-960d-49c272a58794",
		ContentType: libsf.ContentTypeNote,
		Deleted:     false,
		Note: &libsf.Note{
			Title: "The Title",
			Text:  "The text",
		},
		// Internal stuff
		Version:    libsf.ProtocolVersion3,
		AuthParams: libsf.NewAuth("george.abitbol@nowhere.lan", libsf.ProtocolVersion3, "nonce", 420000),
	}

	err := item.Seal(keychain)
	assert.NoError(t, err)

	components := strings.Split(item.EncryptedItemKey, ":")
	assert.Equal(t, 6, len(components))
	assert.Equal(t, libsf.ProtocolVersion3, components[0], "version")
	assert.Regexp(t, "[0-9a-f]{64}", components[1], "auth")
	assert.Equal(t, item.ID, components[2], "item UUID")
	assert.Regexp(t, "[0-9a-f]{32}", components[3], "IV")
	assert.Greater(t, len(components[4]), 0, "note")
	assert.Greater(t, len(components[5]), 0, "auth_params of the user")

	components = strings.Split(item.Content, ":")
	assert.Equal(t, 6, len(components))
	assert.Equal(t, libsf.ProtocolVersion3, components[0], "version")
	assert.Regexp(t, "[0-9a-f]{64}", components[1], "auth")
	assert.Equal(t, item.ID, components[2], "item UUID")
	assert.Regexp(t, "[0-9a-f]{32}", components[3], "IV")
	assert.Greater(t, len(components[4]), 0, "note")
	assert.Greater(t, len(components[5]), 0, "auth_params of the user")
}

func TestItem_Unseal3(t *testing.T) {
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion3,
		MasterKey: "f07d6183b8ea8c50900cf4a767a4d5aeb6fbe821dd6c514f15ec0df8f74d282b",
		AuthKey:   "45318023da0253ac5f28fe3cf48c5a8345df9f720e5fc082c8ba226b28446026",
	}
	item := &libsf.Item{
		ID:               "d989ccc9-15c6-475e-839b-1690bd07d073",
		UserID:           "b329a187-ddf8-4e9b-960d-49c272a58794",
		Content:          "003:d83ea9b696c313c8a352795264873fefebbd60a92c5d9a89e3a380a0d3a68b62:d989ccc9-15c6-475e-839b-1690bd07d073:400591db0ad08c0847f45a1e76ceb87d:HI0PGPWB667YzIlWPR4A8VpGuDb9YOcTRknFb4CZXM2yJ0KK68W2giX6AV6KNV19exUvnunTmkfxlOWUfiG2m7YU2rIO76MMMfs5wqBKqO4eTuootiYVi5JbCW2BFHJcDnj3seb8juBV95Bm5lm4tQ==:eyJpZGVudGlmaWVyIjoiZ2VvcmdlLmFiaXRib2xAbm93aGVyZS5sYW4iLCJ2ZXJzaW9uIjoiMDAzIiwicHdfY29zdCI6NDIwMDAwLCJwd19ub25jZSI6Im5vbmNlIn0=",
		ContentType:      libsf.ContentTypeNote,
		EncryptedItemKey: "003:3c69d9526d2846671c7e8cf89968f3b6ffd92e82ca15b04d29a3f77100ce857c:d989ccc9-15c6-475e-839b-1690bd07d073:93b257d16f53732d81230e41b62eab7c:Ai0xyC1CFcah3/rubAXV+j433oXoABPU8kmYdAzE1WlscKQIXbds8USDG0HmoC1XkCHerozTcJc5IgTAN2JZZBYttmllRswgpn7vDKZIUbXa/FDao3l6a43hedxIfd+4b1moSnB1IgG/T8c+WoA0zDd5vKtB5EMyljLVbyItBZnNrg7toV1bSWQ1t+8xUcKm:eyJpZGVudGlmaWVyIjoiZ2VvcmdlLmFiaXRib2xAbm93aGVyZS5sYW4iLCJ2ZXJzaW9uIjoiMDAzIiwicHdfY29zdCI6NDIwMDAwLCJwd19ub25jZSI6Im5vbmNlIn0=",
		Deleted:          false,
	}

	err := item.Unseal(keychain)

	assert.NoError(t, err)
	assert.Equal(t, "The Title", item.Note.Title)
	assert.Equal(t, "The text", item.Note.Text)
}

func TestItem_SealUnseal3(t *testing.T) {
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion3,
		MasterKey: "f07d6183b8ea8c50900cf4a767a4d5aeb6fbe821dd6c514f15ec0df8f74d282b",
		AuthKey:   "45318023da0253ac5f28fe3cf48c5a8345df9f720e5fc082c8ba226b28446026",
	}
	note := &libsf.Note{
		Title: "The Title",
		Text:  "The text",
	}
	item := &libsf.Item{
		ID:          "d989ccc9-15c6-475e-839b-1690bd07d073",
		UserID:      "b329a187-ddf8-4e9b-960d-49c272a58794",
		ContentType: libsf.ContentTypeNote,
		Deleted:     false,
		Note:        note,
		// Internal stuff
		Version:    libsf.ProtocolVersion3,
		AuthParams: libsf.NewAuth("george.abitbol@nowhere.lan", libsf.ProtocolVersion3, "nonce", 420000),
	}

	err := item.Seal(keychain)
	assert.NoError(t, err)

	item.Note = nil
	assert.Nil(t, item.Note)

	err = item.Unseal(keychain)
	assert.NoError(t, err)

	assert.Equal(t, note.Title, item.Note.Title)
	assert.Equal(t, note.Text, item.Note.Text)
}

//
//
//
//

func TestItem_Unseal4_ItemsKey(t *testing.T) {
	auth := libsf.NewAuth("a@a.lan", libsf.ProtocolVersion4, "66d463427f417c5c8660733ffb1f7a4786d14fe3d7946b5116b5388f9cf80123", 0)
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion4,
		MasterKey: auth.SymmetricKeyPair("12345678").MasterKey,
		ItemsKey:  map[string]string{},
	}
	item := &libsf.Item{
		ID:               "3393636f-959f-4365-aa4c-4c473d6c84e7",
		UserID:           "d26846e8-7669-43fb-a168-253f8a97778a",
		ContentType:      libsf.ContentTypeItemsKey,
		Deleted:          false,
		EncryptedItemKey: "004:0fae489ed14aeca270c4afce2623b84629bb2ffd0a7b5743:c2rJ92J2QNqtDfyP6chOcHWelRnOVWFI1QTZid2gfAYNXtlljPInO86ASco67K16CXQIJfFXorhRZAW43YK62NIudocnz4ScBtHSG5LyNSQ=:eyJrcCI6eyJpZGVudGlmaWVyIjoiYUBhLmxhbiIsInB3X25vbmNlIjoiNjZkNDYzNDI3ZjQxN2M1Yzg2NjA3MzNmZmIxZjdhNDc4NmQxNGZlM2Q3OTQ2YjUxMTZiNTM4OGY5Y2Y4MDEyMyIsInZlcnNpb24iOiIwMDQiLCJvcmlnaW5hdGlvbiI6InJlZ2lzdHJhdGlvbiJ9LCJ1IjoiMzM5MzYzNmYtOTU5Zi00MzY1LWFhNGMtNGM0NzNkNmM4NGU3IiwidiI6IjAwNCJ9",
		Content:          "004:ecac951ca3fb1c383885c3114c70b3af6eae91870d4671e8:koN/pUZpc+EcYy/JOpLBZ79l/LsqFnn1tPAMhZW9XQM0YNu8nvuq/Otiqt5qXx0RZBrdHISiXaabF4MgK3Rw6ExRvvG5QKYP7SMBk617FmmCT3WJhMmwC6vCDLRjpfdr10A642KwLzARyi7+igZ2ro/L+lZeZ3ve2F0D8yEAMK/lqWXHE7fiXW1/TeYmYXCvSNOFlpmVKAf20Dwr04H4CD/mljiQsAaIXTnPzSeL0T/tham6pzkm4fL8DYD3GbJS6QdQhp7BwDxQSHZ7ZUsJzTe5yGAyojrkf5MxqbH+v3mVVnh1:eyJrcCI6eyJpZGVudGlmaWVyIjoiYUBhLmxhbiIsInB3X25vbmNlIjoiNjZkNDYzNDI3ZjQxN2M1Yzg2NjA3MzNmZmIxZjdhNDc4NmQxNGZlM2Q3OTQ2YjUxMTZiNTM4OGY5Y2Y4MDEyMyIsInZlcnNpb24iOiIwMDQiLCJvcmlnaW5hdGlvbiI6InJlZ2lzdHJhdGlvbiJ9LCJ1IjoiMzM5MzYzNmYtOTU5Zi00MzY1LWFhNGMtNGM0NzNkNmM4NGU3IiwidiI6IjAwNCJ9",
		// Internal stuff
		Version:    libsf.ProtocolVersion4,
		AuthParams: auth,
	}

	err := item.Unseal(keychain)
	assert.NoError(t, err)

	assert.Equal(t, keychain.ItemsKey[item.ID], "cfee40a7a0e53eeb0a06d70d46a9c7d69e0c5924fd1b0be7c7d6bedcaa3eeefb")
}

func TestItem_Unseal4_Note(t *testing.T) {
	auth := libsf.NewAuth("a@a.lan", libsf.ProtocolVersion4, "66d463427f417c5c8660733ffb1f7a4786d14fe3d7946b5116b5388f9cf80123", 0)
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion4,
		MasterKey: auth.SymmetricKeyPair("12345678").MasterKey,
		ItemsKey: map[string]string{
			"3393636f-959f-4365-aa4c-4c473d6c84e7": "cfee40a7a0e53eeb0a06d70d46a9c7d69e0c5924fd1b0be7c7d6bedcaa3eeefb",
		},
	}
	item := &libsf.Item{
		ID:               "15035b3d-d03f-4ca4-bdef-c5594047061d",
		UserID:           "d26846e8-7669-43fb-a168-253f8a97778a",
		ContentType:      libsf.ContentTypeNote,
		ItemsKeyID:       "3393636f-959f-4365-aa4c-4c473d6c84e7",
		Deleted:          false,
		EncryptedItemKey: "004:6d0f5ebd7ba9b89a9e90c765599c6c0d7fd6a43e9ff64e0a:aj2EVQ9tm94qczo17hQRHLceTlAHddTMur/7TOSHXOTcDuA/B/bp/dXFMLWdvQpIjK5u7jSTiRlNhT7UuonpMkU1tO5L2qZ4FLBhohWROG8=:eyJ1IjoiMTUwMzViM2QtZDAzZi00Y2E0LWJkZWYtYzU1OTQwNDcwNjFkIiwidiI6IjAwNCJ9",
		Content:          "004:226c5b03350ca61629f54f97db035e3f978dfa06ee7f6746:srs9J6p3KQxhRkfXSu9yytvyR8CkyfSILXtImHOpN2jS9raDpYBXmT7W64ZIorT7SMgNXm3jWPVwgKweRH9KxUzyf8dNd+ezeWAHwqaHzlLBR/7BZulTRFoGryWNMY+t5Agx+6EzCNLiVsnCxICkjsG2rOf8VgZvQH5v1f/RhKm+ZEYaprW3jDfl5p8TuZUYhkHl1BhCz1ARfvFzIliSIYYAL+5S9qcuS3af0CwlN4C9lBN778Y=:eyJ1IjoiMTUwMzViM2QtZDAzZi00Y2E0LWJkZWYtYzU1OTQwNDcwNjFkIiwidiI6IjAwNCJ9",
		// Internal stuff
		Version:    libsf.ProtocolVersion4,
		AuthParams: auth,
	}

	err := item.Unseal(keychain)
	if assert.NoError(t, err) {
		assert.Equal(t, "The Title", item.Note.Title)
		assert.Equal(t, "The text", item.Note.Text)
	}
}

func TestItem_SealUnseal4_Note(t *testing.T) {
	auth := libsf.NewAuth("a@a.lan", libsf.ProtocolVersion4, "66d463427f417c5c8660733ffb1f7a4786d14fe3d7946b5116b5388f9cf80123", 0)
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion4,
		MasterKey: auth.SymmetricKeyPair("12345678").MasterKey,
		ItemsKey: map[string]string{
			"3393636f-959f-4365-aa4c-4c473d6c84e7": "cfee40a7a0e53eeb0a06d70d46a9c7d69e0c5924fd1b0be7c7d6bedcaa3eeefb",
		},
	}
	note := &libsf.Note{
		Title: "The Title",
		Text:  "The text",
	}
	item := &libsf.Item{
		ID:          "15035b3d-d03f-4ca4-bdef-c5594047061d",
		UserID:      "d26846e8-7669-43fb-a168-253f8a97778a",
		ContentType: libsf.ContentTypeNote,
		ItemsKeyID:  "3393636f-959f-4365-aa4c-4c473d6c84e7",
		Deleted:     false,
		Note:        note,
		// Internal stuff
		Version:    libsf.ProtocolVersion4,
		AuthParams: auth,
	}

	err := item.Seal(keychain)
	assert.NoError(t, err)

	item.Note = nil
	assert.Nil(t, item.Note)

	err = item.Unseal(keychain)
	if assert.NoError(t, err) {
		assert.Equal(t, note.Title, item.Note.Title)
		assert.Equal(t, note.Text, item.Note.Text)
	}
}

func TestItem_UnsealSeal4_NoteWithAdditionnalData(t *testing.T) {
	auth := libsf.NewAuth("a@a.lan", libsf.ProtocolVersion4, "66d463427f417c5c8660733ffb1f7a4786d14fe3d7946b5116b5388f9cf80123", 0)
	keychain := &libsf.KeyChain{
		Version:   libsf.ProtocolVersion4,
		MasterKey: auth.SymmetricKeyPair("12345678").MasterKey,
		ItemsKey: map[string]string{
			"3393636f-959f-4365-aa4c-4c473d6c84e7": "cfee40a7a0e53eeb0a06d70d46a9c7d69e0c5924fd1b0be7c7d6bedcaa3eeefb",
		},
	}

	key := "004:6d0f5ebd7ba9b89a9e90c765599c6c0d7fd6a43e9ff64e0a:aj2EVQ9tm94qczo17hQRHLceTlAHddTMur/7TOSHXOTcDuA/B/bp/dXFMLWdvQpIjK5u7jSTiRlNhT7UuonpMkU1tO5L2qZ4FLBhohWROG8=:eyJ1IjoiMTUwMzViM2QtZDAzZi00Y2E0LWJkZWYtYzU1OTQwNDcwNjFkIiwidiI6IjAwNCJ9:eyJ0ZXN0Ijo0Mn0="
	content := "004:226c5b03350ca61629f54f97db035e3f978dfa06ee7f6746:srs9J6p3KQxhRkfXSu9yytvyR8CkyfSILXtImHOpN2jS9raDpYBXmT7W64ZIorT7SMgNXm3jWPVwgKweRH9KxUzyf8dNd+ezeWAHwqaHzlLBR/7BZulTRFoGryWNMY+t5Agx+6EzCNLiVsnCxICkjsG2rOf8VgZvQH5v1f/RhKm+ZEYaprW3jDfl5p8TuZUYhkHl1BhCz1ARfvFzIliSIYYAL+5S9qcuS3af0CwlN4C9lBN778Y=:eyJ1IjoiMTUwMzViM2QtZDAzZi00Y2E0LWJkZWYtYzU1OTQwNDcwNjFkIiwidiI6IjAwNCJ9:eyJ0ZXN0Ijo0Mn0="
	item := &libsf.Item{
		ID:               "15035b3d-d03f-4ca4-bdef-c5594047061d",
		UserID:           "d26846e8-7669-43fb-a168-253f8a97778a",
		ContentType:      libsf.ContentTypeNote,
		ItemsKeyID:       "3393636f-959f-4365-aa4c-4c473d6c84e7",
		Deleted:          false,
		EncryptedItemKey: key,
		Content:          content,
		// Internal stuff
		Version:    libsf.ProtocolVersion4,
		AuthParams: auth,
	}

	err := item.Unseal(keychain)
	if assert.NoError(t, err) {
		assert.Equal(t, "The Title", item.Note.Title)
		assert.Equal(t, "The text", item.Note.Text)
	}

	err = item.Seal(keychain)
	if assert.NoError(t, err) {
		assert.NotEqual(t, key, item.EncryptedItemKey)
		assert.True(t, strings.HasSuffix(item.EncryptedItemKey, ":eyJ0ZXN0Ijo0Mn0="), item.EncryptedItemKey)

		assert.NotEqual(t, content, item.Content)
		assert.True(t, strings.HasSuffix(item.Content, ":eyJ0ZXN0Ijo0Mn0="), item.Content)
	}
}
