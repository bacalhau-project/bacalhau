const path = require("path");
var fs = require("fs");
const os = require("os");
const crypto = require("crypto");

let clientID,
	userIdKey,
	default_url = "http://bootstrap.production.bacalhau.org:1234";

function initializeSDK(url = default_url) {
	// Create config folder
	const homedir = os.homedir();
	const configPath = path.join(homedir, ".bacalhau");
	if (!fs.existsSync(configPath)) {
		fs.mkdirSync(configPath);
	}

	// Create private keys if not exist
	const keyFileName = "user_id.pem";
	const userIdkeyPath = path.join(configPath, keyFileName);
	if (!fs.existsSync(userIdkeyPath)) {
		// Generate Keys
		const { privateKey } = crypto.generateKeyPairSync("rsa", {
			name: "RSASSA-PKCS1-v1_5",
			modulusLength: 2048,
		});

		// Store Private keys
		fs.writeFileSync(
			userIdkeyPath,
			privateKey.export({
				format: "pem",
				type: "pkcs1",
			}),
			{ flag: "w" }
		);
	}

	// Set Private key
	const uIK = loadUserIdKey(userIdkeyPath);
	userIdKey = uIK;

	// Set Client ID
	const cI = loadClientId(userIdkeyPath);
	clientID = cI;

	return { base_url: url };
}

function loadClientId(keyPath) {
	const uIK = loadUserIdKey(keyPath);

	const privateKey = crypto.createPrivateKey({ key: uIK });
	const publicKey = crypto.createPublicKey(privateKey);

	let b = Buffer.from(publicKey.export({ format: "jwk" }).n, "base64");
	let hash = crypto.createHash("sha256");
	hash.update(b);
	return hash.digest("hex");
}

function getClientId() {
	return clientID;
}

function signForClient(obj) {
	// Convert Stringified json data to buffer
	const data = Buffer.from(JSON.stringify(obj));
	// Sign the data and returned signature in buffer
	const sign = crypto.sign("SHA256", data, userIdKey);
	// Convert returned buffer to base64
	const signature = sign.toString("base64");
	// // Printing the signature
	return signature;
}

function getClientPublicKey() {
	const publicKey = crypto.createPublicKey(userIdKey);

	return clean_pem_pub_key(
		publicKey.export({
			format: "pem",
			type: "spki",
		})
	);
}

function loadUserIdKey(keyPath) {
	const privateKey = fs.readFileSync(keyPath, {
		encoding: "utf8",
	});
	return privateKey;
}

function clean_pem_pub_key(str) {
	return str
		.replace(/-----BEGIN PUBLIC KEY-----/, "")
		.replace(/-----END PUBLIC KEY-----/, "")
		.replace(/(\r\n|\n|\r)/gm, "")
		.slice(32);
}

module.exports = {
	initializeSDK,
	getClientPublicKey,
	getClientId,
	signForClient,
};
