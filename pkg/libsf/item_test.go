package libsf_test

import (
	"strings"
	"testing"

	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
)

func TestItem_Seal(t *testing.T) {
	mk := "f07d6183b8ea8c50900cf4a767a4d5aeb6fbe821dd6c514f15ec0df8f74d282b"
	ak := "45318023da0253ac5f28fe3cf48c5a8345df9f720e5fc082c8ba226b28446026"
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
		AuthParams: libsf.NewAuth("george.abitbol@nowhere.lan", "003", "nonce", 420000),
	}

	err := item.Seal(mk, ak)
	assert.NoError(t, err)

	components := strings.Split(item.EncryptedItemKey, ":")
	assert.Equal(t, 6, len(components))
	assert.Equal(t, "003", components[0], "version")
	assert.Regexp(t, "[0-9a-f]{64}", components[1], "auth")
	assert.Equal(t, item.ID, components[2], "item UUID")
	assert.Regexp(t, "[0-9a-f]{32}", components[3], "IV")
	assert.Greater(t, len(components[4]), 0, "note")
	assert.Greater(t, len(components[5]), 0, "auth_params of the user")

	components = strings.Split(item.Content, ":")
	assert.Equal(t, 6, len(components))
	assert.Equal(t, "003", components[0], "version")
	assert.Regexp(t, "[0-9a-f]{64}", components[1], "auth")
	assert.Equal(t, item.ID, components[2], "item UUID")
	assert.Regexp(t, "[0-9a-f]{32}", components[3], "IV")
	assert.Greater(t, len(components[4]), 0, "note")
	assert.Greater(t, len(components[5]), 0, "auth_params of the user")
}

func TestItem_Unseal(t *testing.T) {
	mk := "f07d6183b8ea8c50900cf4a767a4d5aeb6fbe821dd6c514f15ec0df8f74d282b"
	ak := "45318023da0253ac5f28fe3cf48c5a8345df9f720e5fc082c8ba226b28446026"
	item := &libsf.Item{
		ID:               "d989ccc9-15c6-475e-839b-1690bd07d073",
		UserID:           "b329a187-ddf8-4e9b-960d-49c272a58794",
		Content:          "003:d83ea9b696c313c8a352795264873fefebbd60a92c5d9a89e3a380a0d3a68b62:d989ccc9-15c6-475e-839b-1690bd07d073:400591db0ad08c0847f45a1e76ceb87d:HI0PGPWB667YzIlWPR4A8VpGuDb9YOcTRknFb4CZXM2yJ0KK68W2giX6AV6KNV19exUvnunTmkfxlOWUfiG2m7YU2rIO76MMMfs5wqBKqO4eTuootiYVi5JbCW2BFHJcDnj3seb8juBV95Bm5lm4tQ==:eyJpZGVudGlmaWVyIjoiZ2VvcmdlLmFiaXRib2xAbm93aGVyZS5sYW4iLCJ2ZXJzaW9uIjoiMDAzIiwicHdfY29zdCI6NDIwMDAwLCJwd19ub25jZSI6Im5vbmNlIn0=",
		ContentType:      libsf.ContentTypeNote,
		EncryptedItemKey: "003:3c69d9526d2846671c7e8cf89968f3b6ffd92e82ca15b04d29a3f77100ce857c:d989ccc9-15c6-475e-839b-1690bd07d073:93b257d16f53732d81230e41b62eab7c:Ai0xyC1CFcah3/rubAXV+j433oXoABPU8kmYdAzE1WlscKQIXbds8USDG0HmoC1XkCHerozTcJc5IgTAN2JZZBYttmllRswgpn7vDKZIUbXa/FDao3l6a43hedxIfd+4b1moSnB1IgG/T8c+WoA0zDd5vKtB5EMyljLVbyItBZnNrg7toV1bSWQ1t+8xUcKm:eyJpZGVudGlmaWVyIjoiZ2VvcmdlLmFiaXRib2xAbm93aGVyZS5sYW4iLCJ2ZXJzaW9uIjoiMDAzIiwicHdfY29zdCI6NDIwMDAwLCJwd19ub25jZSI6Im5vbmNlIn0=",
		Deleted:          false,
	}

	err := item.Unseal(mk, ak)

	assert.NoError(t, err)
	assert.Equal(t, "The Title", item.Note.Title)
	assert.Equal(t, "The text", item.Note.Text)
}

func TestItem_SealUnseal(t *testing.T) {
	mk := "f07d6183b8ea8c50900cf4a767a4d5aeb6fbe821dd6c514f15ec0df8f74d282b"
	ak := "45318023da0253ac5f28fe3cf48c5a8345df9f720e5fc082c8ba226b28446026"
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
		AuthParams: libsf.NewAuth("george.abitbol@nowhere.lan", "003", "nonce", 420000),
	}

	err := item.Seal(mk, ak)
	assert.NoError(t, err)

	item.Note = nil
	assert.Nil(t, item.Note)

	err = item.Unseal(mk, ak)
	assert.NoError(t, err)

	assert.Equal(t, note.Title, item.Note.Title)
	assert.Equal(t, note.Text, item.Note.Text)
}
