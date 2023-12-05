package omemo

import (
	"context"
	"log"
	"time"

	"mellium.im/communique/internal/client"
)

func SetupClient(c *client.Client, logger *log.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	keyBundleAnnouncementStanza := WrapKeyBundle(c)

	err := c.UnmarshalIQ(ctx, keyBundleAnnouncementStanza.TokenReader(), nil)

	if err != nil {
		logger.Printf("Error sending key bundle: %q", err)
	}

	// ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	// defer cancel()

	// deviceAnnouncementStanza := WrapDeviceIds([]Device{
	// 	{ID: "1", Label: "Acer Aspire 3"},
	// }, c)

	// _, err = c.SendIQ(ctx, deviceAnnouncementStanza.TokenReader())

	// if err != nil {
	// 	logger.Printf("Error sending device list: %q", err)
	// }

}
