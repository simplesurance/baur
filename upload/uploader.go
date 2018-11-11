package upload

//Uploader is an interface for storing files in another place
type Uploader interface {
	Upload(from, to string) (string, error)
}
