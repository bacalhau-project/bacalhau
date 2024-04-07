---
sidebar_label: "Configuring TLS"
sidebar_position: 185
title: "Configuring Transport Level Security"
description: How to configure TLS for the requester node APIs
---

By default, the requester node APIs used by the Bacalhau CLI are accessible over HTTP, but it is possible to configure it to use Transport Level Security (TLS) so that they are accessible over HTTPS instead. There are several ways to obtain the necessary certificates and keys, and Bacalhau supports obtaining them via ACME and Certificate Authorities or even self-signing them.

:::info
Once configured, you must ensure that instead of using **http**://IP:PORT you use **https**://IP:PORT
to access the Bacalhau API
:::

## Getting a certificate from Let's Encrypt with ACME

Automatic Certificate Management Environment (ACME) is a protocol that allows for automating the deployment of Public Key Infrastructure, and is the protocol used to obtain a free certificate from the [Let's Encrypt](https://letsencrypt.org/) Certificate Authority.

Using the `--autocert [hostname]` parameter to the CLI (in the `serve` and `devstack` commands), a certificate is obtained automatically from Lets Encrypt. The provided hostname should be a comma-separated list of hostnames, but they should all be publicly resolvable as Lets Encrypt will attempt to connect to the server to verify ownership (using the [ACME HTTP-01](https://letsencrypt.org/docs/challenge-types/#http-01-challenge) challenge). On the very first request this can take a short time whilst the first certificate is issued, but afterwards they are then cached in the bacalhau repository.

Alternatively, you may set these options via the environment variable, `BACALHAU_AUTO_TLS`. If you are using a configuration file, you can set the values in`Node.ServerAPI.TLS.AutoCert` instead.

:::info
As a result of the Lets Encrypt verification step, it is necessary for the server to be able to handle requests on port 443. This typically requires elevated privileges, and rather than obtain these through a privileged account (such as root), you should instead use setcap to grant the executable the right to bind to ports \<1024.

```
sudo setcap CAP_NET_BIND_SERVICE+ep $(which bacalhau)
```

:::

A cache of ACME data is held in the config repository, by default `~/.bacalhau/autocert-cache`, and this will be used to manage renewals to avoid rate limits.

## Getting a certificate from a Certificate Authority

Obtaining a TLS certificate from a Certificate Authority (CA) without using the Automated Certificate Management Environment (ACME) protocol involves a manual process that typically requires the following steps:

1. Choose a Certificate Authority: First, you need to select a trusted Certificate Authority that issues TLS certificates. Popular CAs include DigiCert, GlobalSign, Comodo (now Sectigo), and others. You may also consider whether you want a free or paid certificate, as CAs offer different pricing models.

2. Generate a Certificate Signing Request (CSR): A CSR is a text file containing information about your organization and the domain for which you need the certificate. You can generate a CSR using various tools or directly on your web server. Typically, this involves providing details such as your organization's name, common name (your domain name), location, and other relevant information.

3. Submit the CSR: Access your chosen CA's website and locate their certificate issuance or order page. You'll typically find an option to "Submit CSR" or a similar option. Paste the contents of your CSR into the provided text box.

4. Verify Domain Ownership: The CA will usually require you to verify that you own the domain for which you're requesting the certificate. They may send an email to one of the standard domain-related email addresses (e.g., admin@yourdomain.com, webmaster@yourdomain.com). Follow the instructions in the email to confirm domain ownership.

5. Complete Additional Verification: Depending on the CA's policies and the type of certificate you're requesting (e.g., Extended Validation or EV certificates), you may need to provide additional documentation to verify your organization's identity. This can include legal documents or phone calls from the CA to confirm your request.

6. Payment and Processing: If you're obtaining a paid certificate, you'll need to make the payment at this stage. Once the CA has received your payment and completed the verification process, they will issue the TLS certificate.

Once you have obtained your certificates, you will need to put two files in a location that bacalhau can read them. You need the server certificate, often called something like `server.cert` or `server.cert.pem`, and the server key which is often called something like `server.key` or `server.key.pem`.

Once you have these two files available, you must start `bacalhau serve` which two new flags. These are `tlscert` and `tlskey` flags, whose arguments should point to the relevant file. An example of how it is used is:

```
bacalhau server --node-type=requester --tlscert=server.cert --tlskey=server.key
```

Alternatively, you may set these options via the environment variables, `BACALHAU_TLS_CERT` and `BACALHAU_TLS_KEY`. If you are using a configuration file, you can set the values in`Node.ServerAPI.TLS.ServerCertificate` and `Node.ServerAPI.TLS.ServerKey` instead.

## Self-signed certificates

If you wish, it is possible to use Bacalhau with a self-signed certificate which does not rely on an external Certificate Authority. This is an involved process and so is not described in detail here although there is [a helpful script in the Bacalhau github repository](https://github.com/bacalhau-project/bacalhau/blob/main/scripts/make-certs.sh) which should provide a good starting point.

Once you have generated the necessary files, the steps are much like above, you must start `bacalhau serve` which two new flags. These are `tlscert` and `tlskey` flags, whose arguments should point to the relevant file. An example of how it is used is:

```
bacalhau server --node-type=requester --tlscert=server.cert --tlskey=server.key
```

Alternatively, you may set these options via the environment variables, `BACALHAU_TLS_CERT` and `BACALHAU_TLS_KEY`. If you are using a configuration file, you can set the values in`Node.ServerAPI.TLS.ServerCertificate` and `Node.ServerAPI.TLS.ServerKey` instead.

If you use self-signed certificates, it is unlikely that any clients will be able to verify the certificate when connecting to the Bacalhau APIs. There are three options available to work around this problem:

1. Provide a CA certificate file of trusted certificate authorities, which many software libraries support in addition to system authorities.

2. Install the CA certificate file in the system keychain of each machine that needs access to the Bacalhau APIs.

3. Instruct the software library you are using not to verify HTTPS requests.
