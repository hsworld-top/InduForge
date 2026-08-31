package gateway

import (
	"io/fs"
	"os"
	"strconv"
)

func osDirFSImpl(root string) fs.FS { return os.DirFS(root) }

func strconvItoa(value int) string { return strconv.Itoa(value) }
