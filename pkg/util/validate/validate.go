package validate

import (
	"github.com/asaskevich/govalidator"
	"strings"
)

func IsDNSHost(str string) bool {
	if str == "" || len(strings.Replace(str, ".", "", -1)) > 63 {
		// constraints already violated
		return false
	}
	return govalidator.IsDNSName(str)
}
