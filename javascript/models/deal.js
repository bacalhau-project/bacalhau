class Deal {
	constructor({ concurrency = 1, confidence = 0, min_bids = 0 } = {}) {
		this.concurrency = concurrency;
		this.confidence = confidence;
		this.min_bids = min_bids;
	}

	get toJson() {
		return {
			concurrency: this.concurrency,
			confidence: this.confidence,
			min_bids: this.min_bids,
		};
	}
}

module.exports = { Deal };
