// Package libreseed - temporary file to initialize dependencies
package libreseed

import (
	_ "github.com/anacrolix/dht/v2"
	_ "github.com/anacrolix/torrent"
	_ "github.com/stretchr/testify/assert"
	_ "gopkg.in/yaml.v3"
)
