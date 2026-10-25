// Browser-synthesised "probable" keyboard sounds for replays. These are not
// recordings of the original keyboard: each profile is a tiny physical model
// of a switch family, rendered with noise, filters and short tones.
export type KeySoundKind = 'key' | 'space' | 'mod' | 'backspace';

interface SoundModel {
	id: string;
	label: string;
	description: string;
	noiseGain: number;
	noiseDecay: number;
	noiseFreq: number;
	noiseQ: number;
	noiseType: BiquadFilterType;
	toneFreq: number;
	toneGain: number;
	toneDecay: number;
	toneType: OscillatorType;
	pingFreq?: number;
	pingGain?: number;
	pingDecay?: number;
}

export const KEY_SOUNDS: SoundModel[] = [
	{
		id: 'membrane',
		label: 'membrane',
		description: 'soft rubber dome · muted thock',
		noiseGain: 0.2,
		noiseDecay: 0.045,
		noiseFreq: 620,
		noiseQ: 0.8,
		noiseType: 'lowpass',
		toneFreq: 105,
		toneGain: 0.14,
		toneDecay: 0.06,
		toneType: 'sine'
	},
	{
		id: 'laptop',
		label: 'laptop scissor',
		description: 'short travel · plastic tick',
		noiseGain: 0.3,
		noiseDecay: 0.022,
		noiseFreq: 2600,
		noiseQ: 1.4,
		noiseType: 'bandpass',
		toneFreq: 420,
		toneGain: 0.08,
		toneDecay: 0.025,
		toneType: 'triangle',
		pingFreq: 3100,
		pingGain: 0.035,
		pingDecay: 0.018
	},
	{
		id: 'red',
		label: 'red switch',
		description: 'linear · smooth bottom-out',
		noiseGain: 0.28,
		noiseDecay: 0.04,
		noiseFreq: 900,
		noiseQ: 0.7,
		noiseType: 'lowpass',
		toneFreq: 150,
		toneGain: 0.12,
		toneDecay: 0.05,
		toneType: 'sine'
	},
	{
		id: 'blue',
		label: 'blue switch',
		description: 'clicky · sharp two-stage click',
		noiseGain: 0.42,
		noiseDecay: 0.028,
		noiseFreq: 2300,
		noiseQ: 1.6,
		noiseType: 'bandpass',
		toneFreq: 1250,
		toneGain: 0.09,
		toneDecay: 0.018,
		toneType: 'square',
		pingFreq: 4200,
		pingGain: 0.05,
		pingDecay: 0.025
	},
	{
		id: 'brown',
		label: 'brown switch',
		description: 'tactile bump · dry clack',
		noiseGain: 0.34,
		noiseDecay: 0.032,
		noiseFreq: 1500,
		noiseQ: 1.1,
		noiseType: 'bandpass',
		toneFreq: 190,
		toneGain: 0.1,
		toneDecay: 0.04,
		toneType: 'triangle'
	},
	{
		id: 'black',
		label: 'black switch',
		description: 'heavy linear · deep thock',
		noiseGain: 0.32,
		noiseDecay: 0.055,
		noiseFreq: 500,
		noiseQ: 0.7,
		noiseType: 'lowpass',
		toneFreq: 85,
		toneGain: 0.2,
		toneDecay: 0.085,
		toneType: 'sine'
	},
	{
		id: 'green',
		label: 'green switch',
		description: 'loud clicky · spring ping',
		noiseGain: 0.48,
		noiseDecay: 0.026,
		noiseFreq: 2700,
		noiseQ: 2,
		noiseType: 'bandpass',
		toneFreq: 1500,
		toneGain: 0.1,
		toneDecay: 0.015,
		toneType: 'square',
		pingFreq: 5600,
		pingGain: 0.06,
		pingDecay: 0.08
	},
	{
		id: 'holy-panda',
		label: 'holy panda',
		description: 'tactile · long spring thock',
		noiseGain: 0.3,
		noiseDecay: 0.05,
		noiseFreq: 480,
		noiseQ: 0.9,
		noiseType: 'lowpass',
		toneFreq: 120,
		toneGain: 0.16,
		toneDecay: 0.075,
		toneType: 'triangle'
	},
	{
		id: 'topre',
		label: 'topre',
		description: 'rubber cup · round thock',
		noiseGain: 0.32,
		noiseDecay: 0.06,
		noiseFreq: 380,
		noiseQ: 0.8,
		noiseType: 'lowpass',
		toneFreq: 72,
		toneGain: 0.2,
		toneDecay: 0.1,
		toneType: 'sine'
	},
	{
		id: 'buckling',
		label: 'buckling spring',
		description: 'vintage ping · long ring',
		noiseGain: 0.4,
		noiseDecay: 0.03,
		noiseFreq: 1900,
		noiseQ: 1.2,
		noiseType: 'bandpass',
		toneFreq: 220,
		toneGain: 0.12,
		toneDecay: 0.05,
		toneType: 'square',
		pingFreq: 2400,
		pingGain: 0.08,
		pingDecay: 0.22
	},
	{
		id: 'typewriter',
		label: 'typewriter',
		description: 'mechanical clack · bell ping',
		noiseGain: 0.5,
		noiseDecay: 0.026,
		noiseFreq: 2100,
		noiseQ: 1.8,
		noiseType: 'bandpass',
		toneFreq: 780,
		toneGain: 0.13,
		toneDecay: 0.035,
		toneType: 'triangle',
		pingFreq: 2900,
		pingGain: 0.05,
		pingDecay: 0.18
	},
	{
		id: 'box-white',
		label: 'box white',
		description: 'crisp click bar · bright snap',
		noiseGain: 0.42,
		noiseDecay: 0.02,
		noiseFreq: 3200,
		noiseQ: 2.2,
		noiseType: 'bandpass',
		toneFreq: 1600,
		toneGain: 0.08,
		toneDecay: 0.012,
		toneType: 'sawtooth'
	},
	{
		id: 'box-jade',
		label: 'box jade',
		description: 'thick click bar · low snap',
		noiseGain: 0.44,
		noiseDecay: 0.03,
		noiseFreq: 1400,
		noiseQ: 1.8,
		noiseType: 'bandpass',
		toneFreq: 600,
		toneGain: 0.12,
		toneDecay: 0.03,
		toneType: 'square'
	},
	{
		id: 'silent',
		label: 'silent linear',
		description: 'dampened · barely there',
		noiseGain: 0.09,
		noiseDecay: 0.026,
		noiseFreq: 700,
		noiseQ: 0.9,
		noiseType: 'lowpass',
		toneFreq: 150,
		toneGain: 0.035,
		toneDecay: 0.03,
		toneType: 'sine'
	},
	{
		id: 'optical',
		label: 'optical',
		description: 'snappy · clean plastic click',
		noiseGain: 0.3,
		noiseDecay: 0.018,
		noiseFreq: 2900,
		noiseQ: 1.5,
		noiseType: 'bandpass',
		toneFreq: 540,
		toneGain: 0.08,
		toneDecay: 0.02,
		toneType: 'triangle',
		pingFreq: 3600,
		pingGain: 0.03,
		pingDecay: 0.02
	},
	{
		id: 'cream',
		label: 'cream linear',
		description: 'lubed · muted smooth thock',
		noiseGain: 0.22,
		noiseDecay: 0.05,
		noiseFreq: 520,
		noiseQ: 0.7,
		noiseType: 'lowpass',
		toneFreq: 110,
		toneGain: 0.14,
		toneDecay: 0.07,
		toneType: 'sine'
	},
	{
		id: 'gasket',
		label: 'gasket mount',
		description: 'soft plate · pillowy thock',
		noiseGain: 0.26,
		noiseDecay: 0.06,
		noiseFreq: 600,
		noiseQ: 0.7,
		noiseType: 'lowpass',
		toneFreq: 130,
		toneGain: 0.1,
		toneDecay: 0.09,
		toneType: 'sine'
	},
	{
		id: 'arcade',
		label: 'arcade cabinet',
		description: 'not a keyboard · pure fun',
		noiseGain: 0.08,
		noiseDecay: 0.02,
		noiseFreq: 1800,
		noiseQ: 1.2,
		noiseType: 'bandpass',
		toneFreq: 880,
		toneGain: 0.1,
		toneDecay: 0.055,
		toneType: 'square',
		pingFreq: 1320,
		pingGain: 0.035,
		pingDecay: 0.04
	}
];

export function soundProfile(id: string): SoundModel {
	return KEY_SOUNDS.find((profile) => profile.id === id) ?? KEY_SOUNDS[0];
}

export class ReplaySoundEngine {
	private ctx: AudioContext | null = null;
	private master: GainNode | null = null;
	private noise: AudioBuffer | null = null;
	private profile: SoundModel = KEY_SOUNDS[0];

	setProfile(id: string) {
		this.profile = soundProfile(id);
	}

	async enable(): Promise<void> {
		if (!this.ctx) {
			const Ctor =
				window.AudioContext ??
				(window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
			if (!Ctor) return;
			this.ctx = new Ctor();
			const compressor = this.ctx.createDynamicsCompressor();
			compressor.threshold.value = -16;
			compressor.knee.value = 16;
			compressor.ratio.value = 6;
			compressor.attack.value = 0.002;
			compressor.release.value = 0.12;
			this.master = this.ctx.createGain();
			this.master.gain.value = 0.34;
			this.master.connect(compressor);
			compressor.connect(this.ctx.destination);
			this.noise = this.makeNoise(this.ctx);
		}
		if (this.ctx.state === 'suspended') {
			await this.ctx.resume();
		}
	}

	disable(): void {
		if (!this.ctx) return;
		void this.ctx.suspend();
	}

	play(kind: KeySoundKind = 'key'): void {
		const ctx = this.ctx;
		const master = this.master;
		const noise = this.noise;
		if (!ctx || !master || !noise || ctx.state !== 'running') return;
		const now = ctx.currentTime + 0.001;
		const model = this.profile;
		const keyScale =
			kind === 'space' ? 0.78 : kind === 'mod' ? 0.9 : kind === 'backspace' ? 0.82 : 1;
		const jitter = 0.92 + Math.random() * 0.16;

		// Noise body: the initial impact of the switch / keycap.
		const source = ctx.createBufferSource();
		source.buffer = noise;
		const filter = ctx.createBiquadFilter();
		filter.type = model.noiseType;
		filter.frequency.value = model.noiseFreq * jitter;
		filter.Q.value = model.noiseQ;
		const body = ctx.createGain();
		const bodyGain = model.noiseGain * keyScale * (0.86 + Math.random() * 0.28);
		body.gain.setValueAtTime(0.0001, now);
		body.gain.exponentialRampToValueAtTime(Math.max(0.0002, bodyGain), now + 0.0015);
		body.gain.exponentialRampToValueAtTime(0.0001, now + model.noiseDecay);
		source.connect(filter);
		filter.connect(body);
		body.connect(master);
		source.start(now);
		source.stop(now + model.noiseDecay + 0.03);

		// Low thock / plastic tone.
		const tone = ctx.createOscillator();
		tone.type = model.toneType;
		tone.frequency.setValueAtTime(model.toneFreq * keyScale * jitter, now);
		tone.frequency.exponentialRampToValueAtTime(
			Math.max(35, model.toneFreq * keyScale * 0.72),
			now + model.toneDecay
		);
		const toneGain = ctx.createGain();
		const toneLevel = model.toneGain * keyScale * (0.9 + Math.random() * 0.2);
		toneGain.gain.setValueAtTime(0.0001, now);
		toneGain.gain.exponentialRampToValueAtTime(Math.max(0.0002, toneLevel), now + 0.002);
		toneGain.gain.exponentialRampToValueAtTime(0.0001, now + model.toneDecay);
		tone.connect(toneGain);
		toneGain.connect(master);
		tone.start(now);
		tone.stop(now + model.toneDecay + 0.02);

		// Optional high ping (springs, typewriter bells, click bars).
		if (model.pingFreq && model.pingGain) {
			const ping = ctx.createOscillator();
			ping.type = 'sine';
			ping.frequency.setValueAtTime(model.pingFreq * jitter, now);
			ping.frequency.exponentialRampToValueAtTime(model.pingFreq * 0.98, now + (model.pingDecay ?? 0.05));
			const pingGain = ctx.createGain();
			pingGain.gain.setValueAtTime(0.0001, now);
			pingGain.gain.exponentialRampToValueAtTime(
				Math.max(0.0002, model.pingGain * keyScale),
				now + 0.002
			);
			pingGain.gain.exponentialRampToValueAtTime(0.0001, now + (model.pingDecay ?? 0.05));
			ping.connect(pingGain);
			pingGain.connect(master);
			ping.start(now);
			ping.stop(now + (model.pingDecay ?? 0.05) + 0.02);
		}
	}

	private makeNoise(ctx: AudioContext): AudioBuffer {
		const length = Math.floor(ctx.sampleRate * 0.25);
		const buffer = ctx.createBuffer(1, length, ctx.sampleRate);
		const data = buffer.getChannelData(0);
		for (let i = 0; i < length; i++) data[i] = Math.random() * 2 - 1;
		return buffer;
	}
}
