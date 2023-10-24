module.exports = {
    "env": {
        "browser": true,
        "es2021": true
    },
    "extends": [
<<<<<<< HEAD
        'next',
        'plugin:@typescript-eslint/recommended',
        'plugin:prettier/recommended',
        "eslint:recommended",
=======
        "eslint:recommended",
        "plugin:@typescript-eslint/recommended",
>>>>>>> 1fa7e1ed (updated lint config)
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
<<<<<<< HEAD
        'prettier',
=======
>>>>>>> 1fa7e1ed (updated lint config)
        "react"
    ],
    "rules": {
    }
}
