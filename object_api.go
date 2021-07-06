package main

// APIObject is an object coming from the API.
type APIObject map[string]interface{}

// Object returns object from API object data.
func (o *APIObject) Object() *Object {
	uid := (*o)["_uid"]
	data := make(map[string]interface{})
	for k, v := range *o {
		switch k {
		case "_uid", "_created", "_author", "_modified", "_modifier":
			{
				break
			}
		default:
			{
				data[k] = v
				break
			}
		}
	}
	if uid == nil {
		uid = ""
	}
	return &Object{
		UID:  uid.(string),
		Data: data,
	}
}