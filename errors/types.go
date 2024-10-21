package errors

func BadRequest(format string, args ...any) *Error {
	return New(400, format, args...)
}

func Unauthorized(format string, args ...any) *Error {
	return New(401, format, args...)
}

func Forbidden(format string, args ...any) *Error {
	return New(403, format, args...)
}

func NotFound(format string, args ...any) *Error {
	return New(404, format, args...)
}

func MethodNotAllowed(format string, args ...any) *Error {
	return New(405, format, args...)
}

func RequestTimeout(format string, args ...any) *Error {
	return New(408, format, args...)
}

func TooManyRequests(format string, args ...any) *Error {
	return New(429, format, args...)
}

func Internal(format string, args ...any) *Error {
	return New(500, format, args...)
}

func NotImplemented(format string, args ...any) *Error {
	return New(501, format, args...)
}

func ServiceUnavailable(format string, args ...any) *Error {
	return New(503, format, args...)
}

func GatewayTimeout(format string, args ...any) *Error {
	return New(504, format, args...)
}
