package tsx

import (
	"errors"
	"os"
)

func GenTs(fp string, rtn []string) error {
	f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	var rtnErr error

	for _, str := range rtn {
		_, err := f.WriteString(str + "\n\n")
		rtnErr = errors.Join(
			rtnErr, err,
		)
	}

	return rtnErr
}
