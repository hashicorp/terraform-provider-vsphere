poll "closed_issue_locker" "locker" {
    schedule = "0 50 13 * * *"
    closed_for = "720h" # 30 days
    max_issues = 50
    sleep_between_issues = "10s"
    message = <<-EOF
    I'm going to lock this issue because it has been closed for _30 days_ ⏳. This helps our maintainers find and focus on the active issues.

    If you feel this issue should be reopened, we encourage creating a new issue linking back to this one for added context. If you feel I made an error 🤖 🙉  , please reach out to my human friends 👉  hashibot-feedback@hashicorp.com. Thanks!
    EOF
}