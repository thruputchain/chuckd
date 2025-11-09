package libtools

import "path"

func JobDir(id string) string {
	return path.Join("/tmp/hogci/job", id)
}
