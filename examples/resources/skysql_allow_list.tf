resource "skysql_allow_list" "default" {
  service_id = skysql_service.default.id
  allow_list = [
    {
      "ip" : "127.0.0.1/32",
      "comment" : "localhost"
    }
  ]
  wait_for_creation = true
}