package storage

import "mime"

func init() {
	// chrome download apk file will be .zip file
	// so we need to set the correct mime type. eg: aws_s3
	mime.AddExtensionType(".apk", "application/vnd.android.package-archive")
}
