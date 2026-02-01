namespace go kitexdemo

# -----------------------------------------------------------------
# Enums
# -----------------------------------------------------------------
enum LogLevel {
    DEBUG = 0,
    INFO  = 1,
    WARN  = 2,
    ERROR = 3,
}

enum UserRole {
    GUEST = 0,
    USER  = 1,
    ADMIN = 2,
}

# -----------------------------------------------------------------
# Structs — covers: required/optional, primitives, enum fields,
#           map, list, nested struct, binary
# -----------------------------------------------------------------
struct EchoRequest {
    1: required string  message
    2: optional i64     timestamp
}

struct EchoResponse {
    1: required string  message
    2: required i64     elapsed_ms
    3: optional string  server_id
}

struct User {
    1: required string             id
    2: required string             name
    3: required i32                age
    4: optional string             email
    5: optional UserRole           role
    6: optional map<string,string> tags
    7: optional list<string>       permissions
}

struct GetUserRequest {
    1: required string user_id
}

struct GetUserResponse {
    1: required User user
}

struct ListUsersRequest {
    1: optional i32    page
    2: optional i32    page_size
    3: optional string filter
}

struct ListUsersResponse {
    1: required list<User> users
    2: required i32        total
}

struct LogEventRequest {
    1: required string             event
    2: required LogLevel           level
    3: optional map<string,string> context_data
}

# -----------------------------------------------------------------
# Exceptions — two distinct exception types to show throws with
#              multiple exception branches
# -----------------------------------------------------------------
exception ServiceException {
    1: required i32    code
    2: required string message
    3: optional string details
}

exception NotFoundException {
    1: required string resource
    2: required string id
}

# -----------------------------------------------------------------
# Service definition
# -----------------------------------------------------------------
service UserService {
    # Unary — basic request/response
    EchoResponse Echo(1: EchoRequest req)
        throws (1: ServiceException err),

    # Unary — demonstrates multiple exception types in throws
    GetUserResponse GetUser(1: GetUserRequest req)
        throws (1: ServiceException err, 2: NotFoundException notFound),

    # Unary — complex return type (list + pagination)
    ListUsersResponse ListUsers(1: ListUsersRequest req)
        throws (1: ServiceException err),

    # Oneway — fire-and-forget, no response expected by caller
    oneway void LogEvent(1: LogEventRequest req),
}
