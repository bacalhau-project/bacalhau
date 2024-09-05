import React, { MouseEvent, useEffect, useState } from "react"
import { toast } from "react-toastify"
import { AuthnRequest, Requirement } from "../../helpers/authInterfaces"
import { bacalhauAPI } from "../../services/bacalhau"
import { AskInput } from "./AskInput"

interface InputFormProps {
    authenticate: (req: AuthnRequest) => void
}

type InputForm = React.ReactElement<InputFormProps>

interface MethodPickerProps {
    setInputForm: (form?: InputForm) => void
    authenticate: InputFormProps['authenticate']
}

// The list of authentication types we currently support, which will be used to
// filter the methods that the user can pick.
const supportedTypes = ["ask"] as const

// The types of requirements that we support.
type supportedRequirement = Extract<Requirement, { type: typeof supportedTypes[number] }>

// Returns whether the passed requirement is supported.
function isSupported(req: Requirement): req is supportedRequirement {
    return (supportedTypes as readonly string[]).includes(req.type)
}

function buttonHintText(supported: boolean, type: string): React.HTMLAttributes<HTMLButtonElement>['title'] {
    return supported ? undefined : `Unsupported authentication type '${type}'`
}

export const MethodPicker: React.FC<MethodPickerProps> = (props) => {
    const [methods, setMethods] = useState<{ [key: string]: Requirement }>({})

    useEffect(() => {
        bacalhauAPI.authMethods()
            .then(methods => { setMethods(methods) })
            .catch(_ => { toast.error("Failed to retrieve authentication methods") })
    }, [])


    // Picks the appropriate input form to respond to this type of requirement.
    const pickInputForm = function (name: string, method: supportedRequirement): InputForm {
        switch (method.type) {
            case "ask":
                return <AskInput
                    name={name}
                    cancel={() => props.setInputForm(undefined)}
                    authenticate={props.authenticate}
                    requirement={method.params} />
            default:
                const _: never = method.type
                throw new Error(`programming error: user was able to select unsupported type ${method.type}`)
        }
    }

    const chooseAuthMethod = (event: MouseEvent<HTMLButtonElement>) => {
        const key = event.currentTarget.dataset["key"] ?? ""
        const method = methods[key]
        if (!isSupported(method))
            throw new Error(`programming error: user was able to select unsupported type ${method.type}`)

        props.setInputForm(pickInputForm(key, method))
    }

    return <div>
        <h3>Pick an authentication method:</h3>
        <ul>
        {Object.keys(methods).map(key => {
            const method = methods[key]
            const supported = isSupported(method)
            return <li key={key}>
                <button
                    disabled={!supported}
                    onClick={chooseAuthMethod}
                    title={buttonHintText(supported, method.type)}
                    data-key={key}>
                    Authenticate using {key.replaceAll(/[^A-Za-z0-9]/g, ' ')}
                </button>
            </li>
        })}
    </ul></div>
}
