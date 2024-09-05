import React, { useState } from "react"
import { toast } from "react-toastify"
import { Layout } from "../../layout/Layout"
import { AuthnRequest } from "../../helpers/authInterfaces"
import { bacalhauAPI } from "../../services/bacalhau"
import { useLocation, useNavigate } from "react-router-dom"
import { AxiosError } from "axios"
import styles from "./Flow.module.scss";
import { MethodPicker } from "./MethodPicker"

export const Flow: React.FC<{}> = ({ }) => {
    const location = useLocation()
    const navigate = useNavigate()
    const [inputForm, setInputForm] = useState<React.ReactElement>()

    const submitAuthnRequest = (req: AuthnRequest) => {
        bacalhauAPI.authenticate(req).then(auth => {
            if (!auth.success) {
                toast.error("Failed to authenticate you: " + auth.reason)
                return
            }

            if ("prev" in location.state && "pathname" in location.state.prev) {
                // If we were navigated here from an auth error on another page,
                // return to that page to continue what we were doing.
                navigate(location.state.prev.pathname)
            } else {
                toast.info("Authentication successful.")
            }
        }).catch(error => {
            let errorText = error
            if (error instanceof AxiosError) {
                errorText = error.response?.statusText
            }
            toast.error("Failed to authenticate you: " + errorText)
        })
    }

    return <Layout pageTitle="Authenticate">
        <div className={styles.flow}>
            {inputForm ?? <MethodPicker
                setInputForm={setInputForm}
                authenticate={submitAuthnRequest} />
            }
        </div>
    </Layout>
}
