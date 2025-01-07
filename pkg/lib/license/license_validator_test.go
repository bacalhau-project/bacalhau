package license

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJWKS = json.RawMessage(`{
	"keys": [
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "key_1",
		  "kty": "RSA",
		  "n": "nh4gGRdy0lHT7hqnK0PDBixYur9d8bJZcGDQlhekphjPY16K4YItDPn6JD_nordyd6vF2QXCFl0qprwgjphzyN-dNAYSGQI81uLl5KPpEVaXAdPQIdknMCcojLEOfctvHhv4GIT-yG7eMr4h-2o-QKnP0qLET2qhaUZT6OcKC4884Fikj7WATvZ5dSZy0dOfoTSylkm1hDJ83WSDGC4biQsTqYHkZpnDlBtwWKMyfCR72QIWqY5OoOb0x4XnJt7imBxHDyifxXw2kMOhUnyMcBMI5aKByrr-T7_MEfcH-Giq1DZvoBksHP1oVm8DSVoarY7OJgRr3FRkO9bss4THyQ",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "key_2",
		  "kty": "RSA",
		  "n": "ncdrbhcWMhRk12jPI7rOZfD6XVxlIPRlH2xHEE5bHQGc5aqSdqj06ykZmB3Jk8NAUHO6PmXAjPV0NnTy0qdhJlP-IjMEei-fVFYxnWj1id-axTURLt_zatZKbl7kHOCt9lCWnKpnhIBxqprU3HVSXiPzxRiCmCKZGjB1LhFqOW3HtxPuXHG6XBUEPmbN9cThsujxb9sDytWL2dYcoaABcDkRmGoq1X53pLf-VBPLREtM_IzE1FqH-j5YkR6BGzRqQr6-e1XvaYxpgTA1G04PmXCKYidsUdnoy94MJ5OVecDE7QLAT7fwlHV0MlWXX3CldTGAk3FFgcwkXVGRiwdfvw",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "expired_key",
		  "kty": "RSA",
		  "n": "rgmteLvRrUfwMS_h00C3n-OTREzdO7OwpHv47_-hAua4UjweP4gYeoMYsyuc5HyZr2nEfzqF3nATG74Bxr8F25eGmcFnLNVlbP-RQ3yItWuMnYj0Yrxzn8rKAhokO_9Jm6Jb7VLHeNUqh7WRKyFG8DQ2CZynlwU06oM3_2gVhbhwv2CoIENR7B5YelFFXiUMqbMx598OD-yk3nN-yJMJ74Vfs-ES2L3czuoaRGYn55L0U6xsQiQQ4jPn2IN1DetRc4dXzmll48m1eTlHFmbbnroKP4Ud4Eoz16KpL-ImJU8ET0Vt-SEAeeI_YziIGLe7-NTU3cKjAK3SL1dDQOlSjQ",
		  "use": "sig"
		}
	]
}`)

var testJWKSFOrInvalidTokens = json.RawMessage(`{
	"keys": [
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "empty_customer_id_key",
		  "kty": "RSA",
		  "n": "jGxr1qaocF5Yp9xn3zHJFrxeNIb0w1IYDHBW_IlzbMLt-gXBSS8hU5vez_n5YMFJruu-5Itou39r8nwvkd__YbgO5CS-nK8hkWs39ho83x3PEWzq2c7qT4mf6k8ODtNv9t7W-nFpKHovAcpLtPEpmLugocyTOYyBjOdjVMUk_q5hRZMdOJnFV2eGy6jnJ9rmXJYyKdmVxFPkBbrxpQtfjSNZnAOSD32-wZEN3f9BW-gCU--CJAR0QF-1_tVjlxkcWLR6klTtCZLVH7VwMtnfgZJHS5p0KAYBhEELbnemjNKtdaj_oTGo_BjT26-NbHXFEauaBppb1UU2d5aYv3gINQ",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "empty_jti_key",
		  "kty": "RSA",
		  "n": "6ZxKuqBNMd8gd9nz-Xeiijsa2B6-VqDasfrFZaL5EIDZrB4IKXjGbiGmNk8GGDIU1cI4LJ9oYVhlrHkRoT5o8iOxBmZkIF1DyYjGUwvMKv46IombIh0aEz389-s8wnisoOXC-LMvjgetWG-cjijmIxxwr4GkGud63j1AGLoHKykahjdj1ypDPY9W7ezAUwPneZdmfCls7OXx0UFc46WYxqz7m-Xpz94oe9EvzRAI547gaJQD69CnERSxxPvOA5SLaqhXODJHToOgbwLCznWu-U1K3nsBR_BNTeC7ZBzxJCfBb59kShQOFk1Q6k8H7nBYTXWatUKAJZLYzxsneQR7TQ",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "empty_license_id_key",
		  "kty": "RSA",
		  "n": "nU1DR70RwQPWx-wiWfTo-K1YAFFrLOzTe33zhkEhGmT0Ei4PCW190EtOuqZhOrJfc0rOjA1-3aqn_I_qipKreJtd6ceEEhg_yuy_uSCqwTbyUJKEk5DEjZoRifMPMKB5iD0xk_YLIX8iGl3uin8MC5JhikgCG5oUBMcxdtnq0rv9-N3vyzAMQAlYwgfBx4OCoxlyaFCEKfS_uHdHNbg9Y9H9jnDx0SLCYswPXtwsOARdX78H_2Dse_t0BCHJNA8DTUYkjHeCI9moVUIyEj1ZmhQm87y-kxwe-RX-JWkh5OotJ-iCDHPHMWKOPCx6TbFn5KzeBVoTKeEIRym6l_oLsQ",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "empty_license_type_key",
		  "kty": "RSA",
		  "n": "4fTCXdNDs2xRysun4BSqsTQ_VmgxDydt3ofr6M1-lSegaVJoB9fVQkpq2HpJIoBJGMgpicT9wR3SZo9jk66Hb48MTPbI9nZ_G8qwB6AHrHttWyc3mFT3WqAlloVuj62uTrndbo7C4fCT1PeGdBjGtexKiklOg_eqzcPSoYHnWjZabtXfR-H94h23y6fCjCZIpTMWULARq8CBVGw32BknLhqjyuxLFuNNCIA7dFrAw3s7EwSRQSJcDXi35ASs37qi5mK3U-5JX5UvP_L3WKWxNTbHV5yXwjr77wYXiW-FNzlaOHG3ozuuJ5AR-GipfXRY0ddnHcDnIBI3T_ZR6ZAV4w",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "empty_subject_key",
		  "kty": "RSA",
		  "n": "vm3UrguEHOhPmw0KTqjkb5IC8C_iNHZfBn5EXtlySGRJnqy_Y_LVNknJINo_1kfCFoD2YStT2ta1lcGmDElophCPQq0aKzShY2bYHhnkzWsdsneIXTm_l6CN2TGz8Lao3qhijm-HEa_t2pPkGZyibtRwNgctYnQPhZz3NXRuSKVExPadU7gBCkAxmYh97pOo1VZGEPEjDhrSYUQrM0e7A1jGiAAqWQBkpTBdgiv66kPskfDW-n6DxqgOB3gLyi5tL7eZdpHpbp17QB7VcmIOKMaJWkhGqzA02zGl_Ox6TiYUStMRpf5n3cdVr4aCwFRwTf3UMG81VCMxDfo_3OoSjw",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "invalid_version_key",
		  "kty": "RSA",
		  "n": "oV7FAyZhv3XTHcnJuVa0hVLbIv3W6tdDupbOzt2NL3Bnxx2KD3XKYIlFEdvFwAdSid1b7x6Pl4ATlELY_FVs87sR2-MCLTkMMvQzuRb3L_YWDJAGHh9gtGMF7wiOy3NmWSrmJ2jP4CmCXtCl6SY0LMuHbkl37nvyx4ud-rbgX7whOroLURkjWX3W8ySVSZqbUNiR2a68lnXSEXJyDsh_7OWZauTGFdYMHK4nJzWGVsBZS5RNS6zCUqEj6jszOvqFbbADypwzYd3G5v3ps55ljnw5rj099DjCCBpBgfNJCdTpE8ed42zULR_XV3uMhVhWI7qNkBdZCDk0BwYq3mfNrw",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "wrong_issuer_key",
		  "kty": "RSA",
		  "n": "khhUnfB1XTkNPz164O4e9xu3rxE5SDCdw6Se7mUb57UBqDS4g7CTuhjcBb7b7RdFw6pIkF3IlGQXM3Xk3mkGTfwDN7JC4CYcRjWkGuJWx4uj1jVsE8BiBNHLFiv9RWbPsGz7KLB9wChau_e_SY7tqOGjjZwgr6H94Bm5lkHgq91JFR-CxZCaka8v196Q4xAuRvaTh3RLtyPzWG9IAsj_1yZAJOA7D-_zk2YCIbmnEL6wbkndTN1TJR6Lfbw3XM_TfPgDn71ixuzn2ZN01j0H-LSJYUZfl4KST7n4oxKcU7oDOOAqlhxYck7cq5UYn_G-CXNojf6LM-LafJW4e5VpTQ",
		  "use": "sig"
		},
		{
		  "alg": "RS256",
		  "e": "AQAB",
		  "kid": "wrong_product_name_key",
		  "kty": "RSA",
		  "n": "_IZJ_hhX71L819CoM9UXievkROu21zhRdIz7b-r8t2JPc3VFrKqaqdynQUUDOrtbfJ4TDxmkSo8DllJ6v20A2xI8u5IyJ4wxW_W1E0bMmKP1yIW8mdJViL1C-jCNSQj28YJ2GOscBllVr2if-uM-AFCh4ts9HuGtIIdegKgfi2t9okyULNBQwnlICvRtQbllLAFKbZygS-m90EL4C29xXf34mIh4o3kNXjLnRuC3jpmALA7SjYX2-JSdJBQGUR6sBnthnK79mSQ5DcOz6XhkxKALgxq9k43w6VntqBKlM4xMcwcPZP0LlFkve-D0GHYctYcoocyAb_CjZAkl8XQJ5Q",
		  "use": "sig"
		}
	]
}`)

// Test tokens
const (
	validRSASignedJWTToken1   = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8xIiwidHlwIjoiSldUIn0.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4NzM1NywiaWF0IjoxNzM2MjkwOTU3LCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcxIjoic29tZXRoaW5nMV92YWx1ZSJ9LCJuYmYiOjE3MzYxMTgxNTcsInByb2R1Y3QiOiJCYWNhbGhhdSIsInN1YiI6ImN1c3RvbWVyXzEifQ.KapDttXfD6WuBrm2OXY2-CNUYI0qFcs7WzgzvGuqU0llZWT9qusKJq5fVFKJbf1Ug9_Bhv9FqvtDQUhJMbasnfkGUiy384WWLsP5V4lLCLZSBVcydd6N5_XR430Q_2oxXGjMv1ZXp_VAUxNTbq15FWG6wNVR88xQZBK1gyZEYe7-uhua-FwTp1LjRF8h3-f0qbtDEVlDiTsZMbIaODqLTwsTrkZT5bqDO4H9u2cqv3d3XDjBn-aLIvgxrHYzPA5Im2nIcXovAFaD-SSHe6vqSfm1SYhBJywFrVwXya5rSLNR7CBS9vy2hUjnGT4Uprt93unEfSqq3znrKtVXbmsYew"
	validRSASignedJWTToken2   = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8yIiwidHlwIjoiSldUIn0.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8yIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8yIiwiZXhwIjoxMDM3NjI4NzM1NywiaWF0IjoxNzM2MjkwOTU3LCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8yIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMiIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8yIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcyIjoic29tZXRoaW5nMl92YWx1ZSJ9LCJuYmYiOjE3MzYxMTgxNTcsInByb2R1Y3QiOiJCYWNhbGhhdSIsInN1YiI6ImN1c3RvbWVyXzIifQ.BnJV8nFHzhSYb04MIwlUSUnsC9I-mT3TleAAmboFfTw_3I75gqxPh2nq_Eyr7pCgbKhB1Tlao5hizznCdCLqw8FQIRo9Efl4vrX0ehT092-C8pNtjOHxeMlXkbROr8iChJsRghhAkCrQWXvNIFyBe03vKri7xCnHGDEMCL7217FBSXYMDMxB9oHs6Lc7Kv4oCizNBk4yYlJ7hl8rngL78HwrjpN7Y91YjXBfGDoKmgOPe_15Pohryjx0CXs7X3pX9dX--z0qFt5S5PYd3RFfG9iBbgB99OJiaEstsAb_RdsvyibkJfeX6GjZoM6LtOGrPb_u-I2yQVgi7rqUFZiG0w"
	expiredRSASignedJWTToken3 = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImV4cGlyZWRfa2V5IiwidHlwIjoiSldUIn0.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxNzM2MjA1MTgzLCJpYXQiOjE3MzYyOTE1ODMsImlzcyI6Imh0dHBzOi8vZXhwYW5zby5pby8iLCJqdGkiOiJsaWNlbnNlXzEiLCJsaWNlbnNlX2lkIjoibGljZW5zZV8xIiwibGljZW5zZV90eXBlIjoicHJvZF90aWVyXzEiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsIm1ldGFkYXRhIjp7InNvbWV0aGluZzEiOiJzb21ldGhpbmcxX3ZhbHVlIn0sIm5iZiI6MTczNjExODc4MywicHJvZHVjdCI6IkJhY2FsaGF1Iiwic3ViIjoiY3VzdG9tZXJfMSJ9.QUKIHKugJ_tLyVyiXak5HO6Wroy1mNWUMECO4D4Kaj3PFZL7Q37IkOSucECT5swHL1L3frQIogCVL8tlwDXb9MJwEs3mUiUiXk3iYTKi68YQqE_PyD8UbezaSUn0xCKvCqugWV_tptpmxSIyvqoGPuLCj35jBBb5qhXotgY150PQlkEG4FhjOkyxNjQQEgYr8a3BvgqgHdm2FoCyS46QBp3TrSNHO4ogUui6qlLLDjVp3WWs_HXNBeEjzDxHjSeAItTxppM_e0hYMI7vzDHg4lOub1UMm-f0bg3ivTnw3Gp5Ht0zc6ScEJ8fuxONNxTYgb5kkAKKzT8YtITYadk5DA"
)

const (
	emptyCustomerIDToken  = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImVtcHR5X2N1c3RvbWVyX2lkX2tleSIsInR5cCI6IkpXVCJ9.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiIiLCJleHAiOjEwMzc2Mjg5OTI4LCJpYXQiOjE3MzYyOTM1MjgsImlzcyI6Imh0dHBzOi8vZXhwYW5zby5pby8iLCJqdGkiOiJsaWNlbnNlXzEiLCJsaWNlbnNlX2lkIjoibGljZW5zZV8xIiwibGljZW5zZV90eXBlIjoicHJvZF90aWVyXzEiLCJsaWNlbnNlX3ZlcnNpb24iOiJ2MSIsIm1ldGFkYXRhIjp7InNvbWV0aGluZzEiOiJzb21ldGhpbmcxX3ZhbHVlIn0sIm5iZiI6MTczNjEyMDcyOCwicHJvZHVjdCI6IkJhY2FsaGF1Iiwic3ViIjoiY3VzdG9tZXJfMSJ9.RoFVlCt0IgL04rs68YWsJNxtjKY3zkBCDAAD3TaBAJV8dduC4DUNwfCqS93Xy9G-vEFDqzh2dIhBsb2rTDsnDz7A1YYferksUz-cFAzvISbzvYGzGEobmvZXpsQTFQ_Iq3MFzrjzh61I_Mv5qA2QjDDqyPVskvUQx_Sl3up_5TbXVCl_57rxFMpiYoCR0q4zxmPFRLKyzo59UXjqlTTCX2vJ0zjZrLGh-fctCFbr3hUU_ZdELfvUcO8biKEPplHvSBel-VYyYEmwhGnzDpBFT7CMLiYJhbbO32dUAaeJ3CKtz0tl3EXyAMl0o-rxTVvWnFnphO4V7lLJN7HxE3RKew"
	emptyJtiToken         = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImVtcHR5X2p0aV9rZXkiLCJ0eXAiOiJKV1QifQ.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTk5NywiaWF0IjoxNzM2MjkzNTk3LCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoiIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcxIjoic29tZXRoaW5nMV92YWx1ZSJ9LCJuYmYiOjE3MzYxMjA3OTcsInByb2R1Y3QiOiJCYWNhbGhhdSIsInN1YiI6ImN1c3RvbWVyXzEifQ.BNdeY4ifXz1F7PUF52RxEHUIasz6NTTL-cbVok6EOlPSdAXikKozGdiZFIdoLxyCh75OTKBLevGq3TDTfcnevfcTJXATM_dQibr_yR2Ke9JblU3YrLM9h87f0SunZ2scZmL22CCDIbIN4kYY9ZBdIxWFVUru0C-T3qLP_0LzL_CrvJNAqWwNPXgLkJADwNupMBQBpT4qOmMOfbC8EvPN89VHshvjXyJCzLI7bcu2Byi5S60QPQq5tmx3A3uwvGfZl3ZF81a546u-j7CCqBdBYFPfJzx7xbZCAYfS20QUN72zSz9hve0Uy096FcndOvE6zrEOvNHUwmI5149yvfSJBA"
	emptyLicenseIDToken   = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImVtcHR5X2xpY2Vuc2VfaWRfa2V5IiwidHlwIjoiSldUIn0.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTg5NywiaWF0IjoxNzM2MjkzNDk3LCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6IiIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcxIjoic29tZXRoaW5nMV92YWx1ZSJ9LCJuYmYiOjE3MzYxMjA2OTcsInByb2R1Y3QiOiJCYWNhbGhhdSIsInN1YiI6ImN1c3RvbWVyXzEifQ.axh_Njc28CcS3KS6Kr5_XP674Hy_t1X7zq8LA2hGvvS8Q8cjLOBN4D71f4NbmTSNGG--xF07VooJhaHV1D_MZ8ftJCk-T3HavvUq7Hg56j71r6WG9nRm8zIpfTN0oaNoDuRliWBqrwDvDQyt0oDz5btJtQ5JrzFyWyIuZH3jygcyU_VKFP7o2yRO9WmVcRbGQRcFLLNBhhaqWs-M62BMJZeYaY_0U54ZIXlK5kn4PXl0JQtScBhdipXAQGFVePnzCJ-6du3V4n5fhXVtFSPPfd_-66ci3PiJZP4IjGrXRDioq2Wd1LcZ_KFkhRlaldHkBPDPsDRcda6wpLmgqOMjxQ"
	emptyLicenseTypeToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImVtcHR5X2xpY2Vuc2VfdHlwZV9rZXkiLCJ0eXAiOiJKV1QifQ.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTkxMywiaWF0IjoxNzM2MjkzNTEzLCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6IiIsImxpY2Vuc2VfdmVyc2lvbiI6InYxIiwibWV0YWRhdGEiOnsic29tZXRoaW5nMSI6InNvbWV0aGluZzFfdmFsdWUifSwibmJmIjoxNzM2MTIwNzEzLCJwcm9kdWN0IjoiQmFjYWxoYXUiLCJzdWIiOiJjdXN0b21lcl8xIn0.BycKgLsf6AwVyiIdrnb33XxzKUqIq0_mtRo18-g_K2aydrO_YBBoGKEm7ZCYKPZaCM3Q0wbotrh4fFKWZ9oqX3yzHblTQBfMnmaYGpjG9FO7hCLHcyeLSnxjuR_qYXdXRT77Sr5PdlTBUY31tBXtv2rlrs55C76hVSJ8o4o_qUiAWIFhRswT_r9R8yY9S25G_4bN-sXapZSck94QfBnIteFizxSJgK3cZoAncOfZRW8CR93aWTWbZt3nESQqYMndKnDY9rZKCkB3hYDK0bb7xozjopu1th8eKRqiGQr-URJNc9anqRdwF8yZOVgtvs-9zutpjniqPA3UhiXu6XUxFg"
	emptySubjectKeyToken  = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImVtcHR5X3N1YmplY3Rfa2V5IiwidHlwIjoiSldUIn0.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTk4NCwiaWF0IjoxNzM2MjkzNTg0LCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcxIjoic29tZXRoaW5nMV92YWx1ZSJ9LCJuYmYiOjE3MzYxMjA3ODQsInByb2R1Y3QiOiJCYWNhbGhhdSIsInN1YiI6IiJ9.umbMJkvE3p4Pl22De_tvMwYpQ5tKBvQPpcQV_NAm2NbyJmLFMCaP-MaP-gDQEl-vUoabp6V0u6pTJUbCBosVdMlc9EGzBCMhG1ifEzfw1QbhLWPoxFqzRLbNOF28g9RLuk4-8MkASPmagr7JTOd7xjJfepqDVnCSeuWzehhM1VA9OMw3YWfxOpY7rgXBf-zujcu9noBNA1ADPrG3WZX_udY02poqyG1wr8nqdT-7d1jnff_Ov3r4sWygmO83CQ2mNVZk_N1lvTVUZsrNjzvMLqiXxHTvt3LdNEgO1yRhA7hpCIOANycIrwqXv2SD0uN7bYbojbidK-u0JbkTjAg5zQ"
	invalidVersionToken   = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImludmFsaWRfdmVyc2lvbl9rZXkiLCJ0eXAiOiJKV1QifQ.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTgzMCwiaWF0IjoxNzM2MjkzNDMwLCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjMzIiwibWV0YWRhdGEiOnsic29tZXRoaW5nMSI6InNvbWV0aGluZzFfdmFsdWUifSwibmJmIjoxNzM2MTIwNjMwLCJwcm9kdWN0IjoiQmFjYWxoYXUiLCJzdWIiOiJjdXN0b21lcl8xIn0.WZZTS18gSSQBDKYQr6fHK15C1tpoQKDqf8Th8KFBfDRR2Sx_jiJ3ON9U6pq-dxqj3Ko0zhdgZy91vx0Gv2u7wxZ54SAOA-Vu8gQMtElNLEMjeJTx345iMy-uXYKt4nbF4blzXZGoKmocv_OisWyN04b0RHn_UVz8nKYD78CwjTEI2iAgwa5moJ9dxjmbHvXGDaV6FG3Dk76CWQ6sxshuRa9nijV9XdF2vaVNgBg3x4g14lHfd3fta9Q9ik9t41zgVEykamOqT9DbNC7G3bX1I7MEYJFNIox-THLv1Ty-nWGnjpq83tAJuhCrKNTuEJuAjpFPZaQx-gJ8NKO1_sPTog"
	wrongIssuerToken      = "eyJhbGciOiJSUzI1NiIsImtpZCI6Indyb25nX2lzc3Vlcl9rZXkiLCJ0eXAiOiJKV1QifQ.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTk0NiwiaWF0IjoxNzM2MjkzNTQ2LCJpc3MiOiJodHRwczovL2V4YW1wbGUuaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcxIjoic29tZXRoaW5nMV92YWx1ZSJ9LCJuYmYiOjE3MzYxMjA3NDYsInByb2R1Y3QiOiJCYWNhbGhhdSIsInN1YiI6ImN1c3RvbWVyXzEifQ.Yno49gv4LpTtFodpHwItRTR_JUUgCOa5BWODNSUaGGOd9U1WVyAO8p-GwImEwgm4DcmeK7W1JCZ6Ij82ctR-8owq7VTD6Cg4jD6E_4gSTWWlsxPGNyPWohXa_IzsmPQJ7FKNadmz4Ux60Edo5rrQe_ESjyumabit7CsTXSfmPZGTqpEun7fI-scu1xZM39X5L5Ghw-GwXODFhIHtLdLaglL2SncCNr-nYw4_Bzil8iwGoQBUVjS9E8tf8gKPZpP_wwQPUYqlDMg4ffsCaz1x3OS-lp7BcKGvhuE5-s4xbhNktVosvOn64-c2OYPOqv1xIjryCdp5UphajgwEyrJVgw"
	wrongProductNameToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6Indyb25nX3Byb2R1Y3RfbmFtZV9rZXkiLCJ0eXAiOiJKV1QifQ.eyJjYXBhYmlsaXRpZXMiOnsidGllcl8xIjoibm8ifSwiY3VzdG9tZXJfaWQiOiJjdXN0b21lcl8xIiwiZXhwIjoxMDM3NjI4OTg1NSwiaWF0IjoxNzM2MjkzNDU1LCJpc3MiOiJodHRwczovL2V4cGFuc28uaW8vIiwianRpIjoibGljZW5zZV8xIiwibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidjEiLCJtZXRhZGF0YSI6eyJzb21ldGhpbmcxIjoic29tZXRoaW5nMV92YWx1ZSJ9LCJuYmYiOjE3MzYxMjA2NTUsInByb2R1Y3QiOiJNZW93Iiwic3ViIjoiY3VzdG9tZXJfMSJ9.cWGUJonGpqF732VQ6V6pf2x76lvcOTq0DOD91Amkg4CIU-Qxd8GvqJMeYhMMCC_a-VTAIA5-0JBEyucDZCLuBVdqDizJ--5q6AFgmHIc6XuqoOJCLE9bDlC5POQM_kZ2Rled-qgGbN9FzOrJ4Spx1NPJINh8r3Tbz9y-rRyoSbJOHwkheMI3teY-SgLYBiHsukZLN9u52Jq6ixV10H7dlA2Au8H8rJhBBomV_bPO3o3QcgDvvCRrX5RpYdotjxgMqeU_w4nF96UlzkuAo3cQkfumSpaCbxSOEDD1zgGACGwuWFaoyubBCidrCbBujyV5szzPthuHNZ95RSk4aS_mjg"
)

func TestNewLicenseValidatorFromJSON(t *testing.T) {
	tests := []struct {
		name      string
		jwks      json.RawMessage
		wantErr   bool
		errString string
	}{
		{
			name:    "Valid JWKS",
			jwks:    testJWKS,
			wantErr: false,
		},
		{
			name:      "Invalid JSON",
			jwks:      json.RawMessage(`{invalid json}`),
			wantErr:   true,
			errString: "invalid JWKS JSON",
		},
		{
			name:      "Empty JSON string",
			jwks:      json.RawMessage(`{}`),
			wantErr:   true,
			errString: "missing 'keys' array in JWKS",
		},
		{
			name:      "Empty JSON raw message",
			jwks:      json.RawMessage(``),
			wantErr:   true,
			errString: "empty JWKS JSON",
		},
		{
			name:      "Empty keys array",
			jwks:      json.RawMessage(`{"keys": []}`),
			wantErr:   true,
			errString: "empty 'keys' array in JWKS",
		},
		{
			name:      "Null keys array",
			jwks:      json.RawMessage(`{"keys": null}`),
			wantErr:   true,
			errString: "missing 'keys' array in JWKS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewLicenseValidatorFromJSON(tt.jwks)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	validator, err := NewLicenseValidatorFromJSON(testJWKS)
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		errString string
	}{
		{
			name:    "Valid RSA token",
			token:   validRSASignedJWTToken1,
			wantErr: false,
		},
		{
			name:    "Valid RSA token 2",
			token:   validRSASignedJWTToken2,
			wantErr: false,
		},
		{
			name:      "Empty token",
			token:     "",
			wantErr:   true,
			errString: "token contains an invalid number of segments",
		},
		{
			name:      "Invalid format",
			token:     "not.a.jwt",
			wantErr:   true,
			errString: "failed to parse token: token is malformed:",
		},
		{
			name:      "Unknown key ID",
			token:     "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8zIiwidHlwIjoiSldUIn0.eyJjdXN0b21lcl9pZCI6ImN1c3RvbWVyXzMiLCJleHAiOjEwMzc2MjQwMTA3LCJmZWF0dXJlcyI6WyJmZWF0dXJlXzMiLCJmZWF0dXJlXzMzIl0sImlhdCI6MTczNjI0MzcwNywibGljZW5zZV9pZCI6ImxpY2Vuc2VfMyIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8zIiwibGljZW5zZV92ZXJzaW9uIjoidl8zIiwibGltaXRhdGlvbnMiOnsidGllcl8zIjoibm8ifSwibWV0YWRhdGEiOnsic29tZXRoaW5nMyI6InNvbWV0aGluZzNfdmFsdWUifSwicHJvZHVjdCI6IkJhY2FsaGF1In0.dzWz7FHKWM0SuVDISzxJ7lfXpXOunrJ01PeRjufvhxGv4g6bGwfKFRjiQYEuwrzst_k1zw0d5XL2VWhhjTpETew7728cubugbiA7222FgLdDk-y2hitEsf_cn-Wd3-da56huBO4tuPZifrT_NEdhbnXzB90Xd6ga3xK-oTsjXniHIj6tdLn9rH4Exp44QYLSj_YTlOm5JMUSWdD70Fnwx5SlWSST1yx5eGTJ71rRTr-tN6Y5_1tywK6a1Tf3iBmW6y4-jA-94zIfvI2wHvmZXen3KRJKra31pKpjjlLPHpqZ3_tVVV7R1sz4PME4sSlh3yhj4oIO-Ixu-eSo1yDWHw",
			wantErr:   true,
			errString: "key not found: kid \"key_3\"",
		},
		{
			name:      "Invalid signature",
			token:     "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtleV8xIiwidHlwIjoiSldUIn0.eyJjdXN0b21lcl9pZCI6ImN1c3RvbWVyXzEiLCJleHAiOjEwMzc2MjQwMTA3LCJmZWF0dXJlcyI6WyJmZWF0dXJlXzEiLCJmZWF0dXJlXzExIl0sImlhdCI6MTczNjI0MzcwNywibGljZW5zZV9pZCI6ImxpY2Vuc2VfMSIsImxpY2Vuc2VfdHlwZSI6InByb2RfdGllcl8xIiwibGljZW5zZV92ZXJzaW9uIjoidl8xIiwibGltaXRhdGlvbnMiOnsidGllcl8xIjoibm8ifSwibWV0YWRhdGEiOnsic29tZXRoaW5nMSI6InNvbWV0aGluZzFfdmFsdWUifSwicHJvZHVjdCI6IkJhY2FsaGF1In0.XyHdItfNma4zkwwwB_M_xgHGqRhTtNmsdPx491msaalfEKAKDYqCMsE6DhL6cKWRqKsXGx27kaBCun1chiYf_yz1rSfMZny-XdakqIg_ENburNFrNSePn-kGhUPmQLzK9JV4Iph2hTWB6dJ8rFqYewDiJ6yfX_AVymmst4OziPmBiPeDcEtjjSR8MEQynRiKUup76fKVgsgXvT-eUHURXOWBcADEw-UvbyKgEt7FB-baZSryReJTyStpA7E64OFB4fNwfk3h70",
			wantErr:   true,
			errString: "token signature is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validator.ValidateToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestNewLicenseValidatorFromFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "Non-existent file",
			filename: "non_existent.json",
			wantErr:  true,
		},
		{
			name:     "Invalid file permissions",
			filename: "/root/test.json", // Assuming no permission to access /root
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewLicenseValidatorFromFile(tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, validator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validator)
			}
		})
	}
}

func TestValidTokenClaims(t *testing.T) {
	validator, err := NewLicenseValidatorFromJSON(testJWKS)
	require.NoError(t, err)

	tests := []struct {
		name     string
		token    string
		validate func(*testing.T, *LicenseClaims)
	}{
		{
			name:  "RSA token claims",
			token: validRSASignedJWTToken1,
			validate: func(t *testing.T, claims *LicenseClaims) {
				assert.Equal(t, "https://expanso.io/", claims.Issuer)
				assert.Equal(t, "license_1", claims.ID)
				assert.Equal(t, "Bacalhau", claims.Product)
				assert.Equal(t, "customer_1", claims.Subject)
				assert.Equal(t, "license_1", claims.LicenseID)
				assert.Equal(t, "prod_tier_1", claims.LicenseType)
				assert.Equal(t, "v1", claims.LicenseVersion)
				assert.Equal(t, "customer_1", claims.CustomerID)
				assert.Equal(t, map[string]string{"tier_1": "no"}, claims.Capabilities)
				assert.Equal(t, map[string]string{"something1": "something1_value"}, claims.Metadata)
				assert.Equal(t, 10376287357, int(claims.ExpiresAt.Unix()))
				assert.Equal(t, 1736118157, int(claims.NotBefore.Unix()))
			},
		},
		{
			name:  "RSA token claims 2",
			token: validRSASignedJWTToken2,
			validate: func(t *testing.T, claims *LicenseClaims) {
				assert.Equal(t, "https://expanso.io/", claims.Issuer)
				assert.Equal(t, "license_2", claims.ID)
				assert.Equal(t, "Bacalhau", claims.Product)
				assert.Equal(t, "customer_2", claims.Subject)
				assert.Equal(t, "license_2", claims.LicenseID)
				assert.Equal(t, "prod_tier_2", claims.LicenseType)
				assert.Equal(t, "v1", claims.LicenseVersion)
				assert.Equal(t, "customer_2", claims.CustomerID)
				assert.Equal(t, map[string]string{"tier_2": "no"}, claims.Capabilities)
				assert.Equal(t, map[string]string{"something2": "something2_value"}, claims.Metadata)
				assert.Equal(t, 10376287357, int(claims.ExpiresAt.Unix()))
				assert.Equal(t, 1736118157, int(claims.NotBefore.Unix()))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validator.ValidateToken(tt.token)
			require.NoError(t, err)
			require.NotNil(t, claims)
			tt.validate(t, claims)
		})
	}
}

func TestExpiredTokenVerification(t *testing.T) {
	validator, err := NewLicenseValidatorFromJSON(testJWKS)
	assert.NoError(t, err)

	claims, err := validator.ValidateToken(expiredRSASignedJWTToken3)
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestValidateAdditionalConstraints(t *testing.T) {
	validator, err := NewLicenseValidatorFromJSON(testJWKSFOrInvalidTokens)
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		errString string
	}{
		{
			name:      "Invalid version",
			token:     invalidVersionToken,
			wantErr:   true,
			errString: "unsupported license version",
		},
		{
			name:      "Wrong product name",
			token:     wrongProductNameToken,
			wantErr:   true,
			errString: "invalid product: expected 'Bacalhau'",
		},
		{
			name:      "Empty license_id",
			token:     emptyLicenseIDToken,
			wantErr:   true,
			errString: "license_id is required",
		},
		{
			name:      "Empty license_type",
			token:     emptyLicenseTypeToken,
			wantErr:   true,
			errString: "license_type is required",
		},
		{
			name:      "Empty customer_id",
			token:     emptyCustomerIDToken,
			wantErr:   true,
			errString: "customer_id is required",
		},
		{
			name:      "Wrong issuer",
			token:     wrongIssuerToken,
			wantErr:   true,
			errString: "invalid issuer: expected 'https://expanso.io/'",
		},
		{
			name:      "Empty subject",
			token:     emptySubjectKeyToken,
			wantErr:   true,
			errString: "subject is required",
		},
		{
			name:      "Empty jti",
			token:     emptyJtiToken,
			wantErr:   true,
			errString: "jti is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := validator.ValidateToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errString != "" {
					assert.Contains(t, err.Error(), tt.errString)
				}
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}
