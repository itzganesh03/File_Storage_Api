package constants

// HTTP status messages
const (
	StatusSuccess               = "success"
	StatusError                 = "error"
	MessageUserCreated          = "User created successfully"
	MessageUserExists           = "User already exists"
	MessageInvalidCredentials   = "Invalid username or password"
	MessageUserNotFound         = "User not found"
	MessageFileUploaded         = "File uploaded successfully"
	MessageFileDeleted          = "File deleted successfully"
	MessageFileNotFound         = "File not found"
	MessageFileDuplicate        = "File with the same name already exists"
	MessageStorageLimitExceeded = "Storage limit exceeded"
	MessageInvalidToken         = "Invalid or expired token"
	MessageUnauthorized         = "Unauthorized access"
	MessageInvalidRequest       = "Invalid request format"
)

// Default values
const (
	DefaultStoragePerUser = 104857600 // 100MB in bytes
	DefaultJWTExpiration  = 24        // hours	DefaultPort           = 8080
	DefaultHost           = "localhost"
	ConfigFilePath        = "conf/config.yml"
)

// Header constants
const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
)

// Database related constants
const (
	MongoDBUserIDField       = "_id"
	MongoDBUsernameField     = "username"
	MongoDBPasswordField     = "password"
	MongoDBStorageLimitField = "storage_limit"
	MongoDBStorageUsedField  = "storage_used"
	MongoDBCreatedAtField    = "created_at"
	MongoDBUpdatedAtField    = "updated_at"
)

// File related constants
const (
	MaxMultipartMemory = 32 << 20 // 32MB
)
