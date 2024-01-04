{
    "Name": "Python",
    "Namespace": "default",
    "Type": "batch",
    "Count": 1,
    "Tasks": [
        {
            "Name": "execute",
            "Engine": {
                "Type": "python",
                "Params": {
                    "Version": "{{or (index . "version") "3.11"}}"
                }
            }
        }
    ]
}
