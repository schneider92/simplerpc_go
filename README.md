# SimpleRPC go
This library is the server-side implementation of simpleRPC for golang.

# Types
The primitive types supported by simpleRPC are mapped to go types as follows:
* Integer => int64
* String => string
* Blob => []byte

The package provides facilities for serializing/deserializing these primitive tpyes. The following functions are present:
* SerializeInteger
* SerializeBlob
* SerializeString
* DeserializeInteger
* DeserializeBlob
* DeserializeString
The serializing functions take an input buffer that it appends the value to and the value to append, and return a new buffer with the value appended. The deserializing functions take an input buffer, extract the value, then returns the remaining buffer (for deserializing the rest of the data) and the deserialized value.

# Server
The package provides the ```Server``` type for accessing the server features of the library. This type can be instantiated using the ```NewServer``` function which accepts a slice of the server services. The services are automatically generated from the service files.

Once you have a ```Server``` instance, you can call ```ProcessRequest``` on it, and that will call the requested function on the requested service.
