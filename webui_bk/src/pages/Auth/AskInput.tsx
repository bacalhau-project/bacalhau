import React from "react";
import { JSONSchema7, JSONSchema7Definition } from "json-schema";
import { AskRequest, AskRequirement, AuthnRequest } from "../../helpers/authInterfaces";
import "./AskInput.module.scss"

interface AskInputProps {
    name: string
    requirement: AskRequirement
    authenticate: (req: AuthnRequest) => void
    cancel: () => void
}

function propertyToInputType(property: JSONSchema7Definition): React.HTMLInputTypeAttribute {
    if (typeof property === "boolean") {
        return "text"
    }

    if (property.writeOnly) {
        return "password"
    }

    switch (property.type) {
        case "number":
        case "integer":
            return "number"
        case "boolean":
            return "checkbox"
        default:
            return "text"
    }
}

export const AskInput: React.FC<AskInputProps> = (props: AskInputProps) => {
    const requirement = props.requirement
    const properties = requirement.properties ?? {}
    const fields = Object.keys(properties)
    const required = requirement.required ?? []

    // Sort by fields listed in required order
    fields.sort((a, b) => required.indexOf(a) - required.indexOf(b))

    const inputs = fields.map(field => {
        const property = properties[field] as JSONSchema7
        return <>
            <label htmlFor={field} data-required={required.includes(field)}>
                {field}
            </label>
            <input
                name={field}
                type={propertyToInputType(property)}
                required={required.includes(field)} />
        </>
    })

    const submit: React.FormEventHandler<HTMLFormElement> = (event) => {
        event.preventDefault()

        const formData = new FormData(event.currentTarget)
        const objectData: AskRequest = {}
        formData.forEach((value, key) => {objectData[key] = value.toString()})

        const request: AuthnRequest = {Name: props.name, MethodData: objectData}
        props.authenticate(request)

        return false
    }

    return <form onSubmit={submit} onReset={props.cancel}>
        <h3>Authenticate using {props.name.replaceAll(/[^A-Za-z0-9]/g, ' ')}</h3>
        {...inputs}
        <input type="reset" value="Cancel"/>
        <input type="submit" value="Authenticate"/>
    </form>
}
