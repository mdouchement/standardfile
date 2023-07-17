//
// libsf is client that interacts with StandardFile/StandardNotes API for syncing encrypted notes.
//

// Create client
//
//	client, err := libsf.NewDefaultClient("https://notes.nas.lan")
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Authenticate
//
//	email := "george.abitbol@nas.lan"
//	password := "12345678"
//
//	auth, err := client.GetAuthParams(email)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	err = auth.IntegrityCheck()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create keychain containing all the keys used for encryption and authentication.
//	keychain := auth.SymmetricKeyPair(password)
//
//	err = client.Login(auth.Email(), keychain.Password)
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Get all items
//
//	items := libsf.NewSyncItems()
//	items, err = client.SyncItems(items) // No sync_token and limit are setted so we get all items.
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Append `SN|ItemsKey` to the KeyChain.
//	for _, item := range items.Retrieved {
//		if item.ContentType != libsf.ContentTypeItemsKey {
//			continue
//		}
//
//		err = item.Unseal(keychain)
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
//	var last int
//	for i, item := range items.Retrieved {
//		switch item.ContentType {
//		case libsf.ContentTypeUserPreferences:
//			// Unseal Preferences item using keychain.
//			err = item.Unseal(keychain)
//			if err != nil {
//				log.Fatal(err)
//			}
//
//			// Parse metadata.
//			if err = item.Note.ParseRaw(); err != nil {
//				log.Fatal(err)
//			}
//
//			fmt.Println("Items are sorted by:", item.Note.GetSortingField())
//		case libsf.ContentTypeNote:
//			// Unseal Note item using keychain.
//			err = item.Unseal(keychain)
//			if err != nil {
//				log.Fatal(err)
//			}
//
//			// Parse metadata.
//			if err = item.Note.ParseRaw(); err != nil {
//				log.Fatal(err)
//			}
//
//			fmt.Println("Title:", item.Note.Title)
//			fmt.Println("Content:", item.Note.Text)
//
//			last = i
//		}
//	}
//
// Update an item
//
//	item := items.Retrieved[last]
//	item.Note.Title += " updated"
//	item.Note.Text += " updated"
//
//	item.Note.SetUpdatedAtNow()
//	item.Note.SaveRaw()
//
//	err = item.Seal(keychain)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Syncing updated item.
//	items = libsf.NewSyncItems()
//	items.Items = append(items.Items, item)
//	items, err = client.SyncItems(items)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	if len(items.Conflicts) > 0 {
//		log.Fatal("items conflict")
//	}
//	fmt.Println("Updated!")
package libsf
