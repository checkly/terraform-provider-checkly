# Simple Enviroment Variable example
resource "checkly_environment_variable" "variable-1" {
  key    = "API_KEY"
  value  = "loZd9hOGHDUrGvmW"
  locked = true
}

resource "checkly_environment_variable" "variable-2" {
  key   = "API_URL"
  value = "http://localhost:3000"
}
