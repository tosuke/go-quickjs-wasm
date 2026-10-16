group "default" {
  targets = ["generate"]
}

target "generate" {
  context = "."
  dockerfile = "internal/generate/Dockerfile"
  target = "artifacts"
  output = ["type=local,dest=."]
}
