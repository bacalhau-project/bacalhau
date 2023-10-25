module.exports = {
    "env": {
        "browser": true,
        "es2021": true
    },
    "extends": [
        'next',
        'plugin:@typescript-eslint/recommended',
        'plugin:prettier/recommended',
        "eslint:recommended",
        "plugin:react/recommended"
    ],
    "overrides": [
        {
            "env": {
                "node": true
            },
            "files": [
                ".eslintrc.{js,cjs}"
            ],
            "parserOptions": {
                "sourceType": "script"
            }
        }
    ],
    "parser": "@typescript-eslint/parser",
    "parserOptions": {
        "ecmaVersion": "latest"
    },
    "plugins": [
        "@typescript-eslint",
        'prettier',
        "react"
    ],
    "rules": {
    }
}
