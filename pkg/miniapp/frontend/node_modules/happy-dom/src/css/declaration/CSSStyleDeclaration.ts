import Element from '../../nodes/element/Element.js';
import CSSRule from '../CSSRule.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import CSSStyleDeclarationPropertyManager from './property-manager/CSSStyleDeclarationPropertyManager.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import CSSStyleDeclarationComputedStyle from './computed-style/CSSStyleDeclarationComputedStyle.js';

/**
 * CSS Style Declaration.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/CSSStyleDeclaration
 */
export default class CSSStyleDeclaration {
	// Public properties
	public readonly parentRule: CSSRule = null;

	// Internal properties
	public [PropertySymbol.window]: BrowserWindow;

	// Private properties
	#element: Element;
	#computed: boolean;
	#cache: {
		attributeValue: string | null;
		propertyManager: CSSStyleDeclarationPropertyManager | null;
	} = {
		attributeValue: null,
		propertyManager: null
	};

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 * @param window Window.
	 * @param [options] Options.
	 * @param [options.element] Element.
	 * @param [options.computed] Computed.
	 */
	constructor(
		illegalConstructorSymbol: Symbol,
		window: BrowserWindow,
		options?: {
			element?: Element;
			computed?: boolean;
		}
	) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}
		this[PropertySymbol.window] = window;
		this.#element = options?.element;
		this.#computed = options?.element ? !!options?.computed : false;
	}

	/**
	 * Index properties
	 */

	/* eslint-disable jsdoc/require-jsdoc */

	public get 0(): string {
		return this.item(0) || undefined;
	}

	public get 1(): string {
		return this.item(1) || undefined;
	}

	public get 2(): string {
		return this.item(2) || undefined;
	}

	public get 3(): string {
		return this.item(3) || undefined;
	}

	public get 4(): string {
		return this.item(4) || undefined;
	}

	public get 5(): string {
		return this.item(5) || undefined;
	}

	public get 6(): string {
		return this.item(6) || undefined;
	}

	public get 7(): string {
		return this.item(7) || undefined;
	}

	public get 8(): string {
		return this.item(8) || undefined;
	}

	public get 9(): string {
		return this.item(9) || undefined;
	}

	public get 10(): string {
		return this.item(10) || undefined;
	}

	public get 11(): string {
		return this.item(11) || undefined;
	}

	public get 12(): string {
		return this.item(12) || undefined;
	}

	public get 13(): string {
		return this.item(13) || undefined;
	}

	public get 14(): string {
		return this.item(14) || undefined;
	}

	public get 15(): string {
		return this.item(15) || undefined;
	}

	public get 16(): string {
		return this.item(16) || undefined;
	}

	public get 17(): string {
		return this.item(17) || undefined;
	}

	public get 18(): string {
		return this.item(18) || undefined;
	}

	public get 19(): string {
		return this.item(19) || undefined;
	}

	public get 20(): string {
		return this.item(20) || undefined;
	}

	public get 21(): string {
		return this.item(21) || undefined;
	}

	public get 22(): string {
		return this.item(22) || undefined;
	}

	public get 23(): string {
		return this.item(23) || undefined;
	}

	public get 24(): string {
		return this.item(24) || undefined;
	}

	public get 25(): string {
		return this.item(25) || undefined;
	}

	public get 26(): string {
		return this.item(26) || undefined;
	}

	public get 27(): string {
		return this.item(27) || undefined;
	}

	public get 28(): string {
		return this.item(28) || undefined;
	}

	public get 29(): string {
		return this.item(29) || undefined;
	}

	public get 30(): string {
		return this.item(30) || undefined;
	}

	public get 31(): string {
		return this.item(31) || undefined;
	}

	public get 32(): string {
		return this.item(32) || undefined;
	}

	public get 33(): string {
		return this.item(33) || undefined;
	}

	public get 34(): string {
		return this.item(34) || undefined;
	}

	public get 35(): string {
		return this.item(35) || undefined;
	}

	public get 36(): string {
		return this.item(36) || undefined;
	}

	public get 37(): string {
		return this.item(37) || undefined;
	}

	public get 38(): string {
		return this.item(38) || undefined;
	}

	public get 39(): string {
		return this.item(39) || undefined;
	}

	public get 40(): string {
		return this.item(40) || undefined;
	}

	public get 41(): string {
		return this.item(41) || undefined;
	}

	public get 42(): string {
		return this.item(42) || undefined;
	}

	public get 43(): string {
		return this.item(43) || undefined;
	}

	public get 44(): string {
		return this.item(44) || undefined;
	}

	public get 45(): string {
		return this.item(45) || undefined;
	}

	public get 46(): string {
		return this.item(46) || undefined;
	}

	public get 47(): string {
		return this.item(47) || undefined;
	}

	public get 48(): string {
		return this.item(48) || undefined;
	}

	public get 49(): string {
		return this.item(49) || undefined;
	}

	public get 50(): string {
		return this.item(50) || undefined;
	}

	public get 51(): string {
		return this.item(51) || undefined;
	}

	public get 52(): string {
		return this.item(52) || undefined;
	}

	public get 53(): string {
		return this.item(53) || undefined;
	}

	public get 54(): string {
		return this.item(54) || undefined;
	}

	public get 55(): string {
		return this.item(55) || undefined;
	}

	public get 56(): string {
		return this.item(56) || undefined;
	}

	public get 57(): string {
		return this.item(57) || undefined;
	}

	public get 58(): string {
		return this.item(58) || undefined;
	}

	public get 59(): string {
		return this.item(59) || undefined;
	}

	public get 60(): string {
		return this.item(60) || undefined;
	}

	public get 61(): string {
		return this.item(61) || undefined;
	}

	public get 62(): string {
		return this.item(62) || undefined;
	}

	public get 63(): string {
		return this.item(63) || undefined;
	}

	public get 64(): string {
		return this.item(64) || undefined;
	}

	public get 65(): string {
		return this.item(65) || undefined;
	}

	public get 66(): string {
		return this.item(66) || undefined;
	}

	public get 67(): string {
		return this.item(67) || undefined;
	}

	public get 68(): string {
		return this.item(68) || undefined;
	}

	public get 69(): string {
		return this.item(69) || undefined;
	}

	public get 70(): string {
		return this.item(70) || undefined;
	}

	public get 71(): string {
		return this.item(71) || undefined;
	}

	public get 72(): string {
		return this.item(72) || undefined;
	}

	public get 73(): string {
		return this.item(73) || undefined;
	}

	public get 74(): string {
		return this.item(74) || undefined;
	}

	public get 75(): string {
		return this.item(75) || undefined;
	}

	public get 76(): string {
		return this.item(76) || undefined;
	}

	public get 77(): string {
		return this.item(77) || undefined;
	}

	public get 78(): string {
		return this.item(78) || undefined;
	}

	public get 79(): string {
		return this.item(79) || undefined;
	}

	public get 80(): string {
		return this.item(80) || undefined;
	}

	public get 81(): string {
		return this.item(81) || undefined;
	}

	public get 82(): string {
		return this.item(82) || undefined;
	}

	public get 83(): string {
		return this.item(83) || undefined;
	}

	public get 84(): string {
		return this.item(84) || undefined;
	}

	public get 85(): string {
		return this.item(85) || undefined;
	}

	public get 86(): string {
		return this.item(86) || undefined;
	}

	public get 87(): string {
		return this.item(87) || undefined;
	}

	public get 88(): string {
		return this.item(88) || undefined;
	}

	public get 89(): string {
		return this.item(89) || undefined;
	}

	public get 90(): string {
		return this.item(90) || undefined;
	}

	public get 91(): string {
		return this.item(91) || undefined;
	}

	public get 92(): string {
		return this.item(92) || undefined;
	}

	public get 93(): string {
		return this.item(93) || undefined;
	}

	public get 94(): string {
		return this.item(94) || undefined;
	}

	public get 95(): string {
		return this.item(95) || undefined;
	}

	public get 96(): string {
		return this.item(96) || undefined;
	}

	public get 97(): string {
		return this.item(97) || undefined;
	}

	public get 98(): string {
		return this.item(98) || undefined;
	}

	public get 99(): string {
		return this.item(99) || undefined;
	}

	public get 100(): string {
		return this.item(100) || undefined;
	}

	public get 101(): string {
		return this.item(101) || undefined;
	}

	public get 102(): string {
		return this.item(102) || undefined;
	}

	public get 103(): string {
		return this.item(103) || undefined;
	}

	public get 104(): string {
		return this.item(104) || undefined;
	}

	public get 105(): string {
		return this.item(105) || undefined;
	}

	public get 106(): string {
		return this.item(106) || undefined;
	}

	public get 107(): string {
		return this.item(107) || undefined;
	}

	public get 108(): string {
		return this.item(108) || undefined;
	}

	public get 109(): string {
		return this.item(109) || undefined;
	}

	public get 110(): string {
		return this.item(110) || undefined;
	}

	public get 111(): string {
		return this.item(111) || undefined;
	}

	public get 112(): string {
		return this.item(112) || undefined;
	}

	public get 113(): string {
		return this.item(113) || undefined;
	}

	public get 114(): string {
		return this.item(114) || undefined;
	}

	public get 115(): string {
		return this.item(115) || undefined;
	}

	public get 116(): string {
		return this.item(116) || undefined;
	}

	public get 117(): string {
		return this.item(117) || undefined;
	}

	public get 118(): string {
		return this.item(118) || undefined;
	}

	public get 119(): string {
		return this.item(119) || undefined;
	}

	public get 120(): string {
		return this.item(120) || undefined;
	}

	public get 121(): string {
		return this.item(121) || undefined;
	}

	public get 122(): string {
		return this.item(122) || undefined;
	}

	public get 123(): string {
		return this.item(123) || undefined;
	}

	public get 124(): string {
		return this.item(124) || undefined;
	}

	public get 125(): string {
		return this.item(125) || undefined;
	}

	public get 126(): string {
		return this.item(126) || undefined;
	}

	public get 127(): string {
		return this.item(127) || undefined;
	}

	public get 128(): string {
		return this.item(128) || undefined;
	}

	public get 129(): string {
		return this.item(129) || undefined;
	}

	public get 130(): string {
		return this.item(130) || undefined;
	}

	public get 131(): string {
		return this.item(131) || undefined;
	}

	public get 132(): string {
		return this.item(132) || undefined;
	}

	public get 133(): string {
		return this.item(133) || undefined;
	}

	public get 134(): string {
		return this.item(134) || undefined;
	}

	public get 135(): string {
		return this.item(135) || undefined;
	}

	public get 136(): string {
		return this.item(136) || undefined;
	}

	public get 137(): string {
		return this.item(137) || undefined;
	}

	public get 138(): string {
		return this.item(138) || undefined;
	}

	public get 139(): string {
		return this.item(139) || undefined;
	}

	public get 140(): string {
		return this.item(140) || undefined;
	}

	public get 141(): string {
		return this.item(141) || undefined;
	}

	public get 142(): string {
		return this.item(142) || undefined;
	}

	public get 143(): string {
		return this.item(143) || undefined;
	}

	public get 144(): string {
		return this.item(144) || undefined;
	}

	public get 145(): string {
		return this.item(145) || undefined;
	}

	public get 146(): string {
		return this.item(146) || undefined;
	}

	public get 147(): string {
		return this.item(147) || undefined;
	}

	public get 148(): string {
		return this.item(148) || undefined;
	}

	public get 149(): string {
		return this.item(149) || undefined;
	}

	public get 150(): string {
		return this.item(150) || undefined;
	}

	public get 151(): string {
		return this.item(151) || undefined;
	}

	public get 152(): string {
		return this.item(152) || undefined;
	}

	public get 153(): string {
		return this.item(153) || undefined;
	}

	public get 154(): string {
		return this.item(154) || undefined;
	}

	public get 155(): string {
		return this.item(155) || undefined;
	}

	public get 156(): string {
		return this.item(156) || undefined;
	}

	public get 157(): string {
		return this.item(157) || undefined;
	}

	public get 158(): string {
		return this.item(158) || undefined;
	}

	public get 159(): string {
		return this.item(159) || undefined;
	}

	public get 160(): string {
		return this.item(160) || undefined;
	}

	public get 161(): string {
		return this.item(161) || undefined;
	}

	public get 162(): string {
		return this.item(162) || undefined;
	}

	public get 163(): string {
		return this.item(163) || undefined;
	}

	public get 164(): string {
		return this.item(164) || undefined;
	}

	public get 165(): string {
		return this.item(165) || undefined;
	}

	public get 166(): string {
		return this.item(166) || undefined;
	}

	public get 167(): string {
		return this.item(167) || undefined;
	}

	public get 168(): string {
		return this.item(168) || undefined;
	}

	public get 169(): string {
		return this.item(169) || undefined;
	}

	public get 170(): string {
		return this.item(170) || undefined;
	}

	public get 171(): string {
		return this.item(171) || undefined;
	}

	public get 172(): string {
		return this.item(172) || undefined;
	}

	public get 173(): string {
		return this.item(173) || undefined;
	}

	public get 174(): string {
		return this.item(174) || undefined;
	}

	public get 175(): string {
		return this.item(175) || undefined;
	}

	public get 176(): string {
		return this.item(176) || undefined;
	}

	public get 177(): string {
		return this.item(177) || undefined;
	}

	public get 178(): string {
		return this.item(178) || undefined;
	}

	public get 179(): string {
		return this.item(179) || undefined;
	}

	public get 180(): string {
		return this.item(180) || undefined;
	}

	public get 181(): string {
		return this.item(181) || undefined;
	}

	public get 182(): string {
		return this.item(182) || undefined;
	}

	public get 183(): string {
		return this.item(183) || undefined;
	}

	public get 184(): string {
		return this.item(184) || undefined;
	}

	public get 185(): string {
		return this.item(185) || undefined;
	}

	public get 186(): string {
		return this.item(186) || undefined;
	}

	public get 187(): string {
		return this.item(187) || undefined;
	}

	public get 188(): string {
		return this.item(188) || undefined;
	}

	public get 189(): string {
		return this.item(189) || undefined;
	}

	public get 190(): string {
		return this.item(190) || undefined;
	}

	public get 191(): string {
		return this.item(191) || undefined;
	}

	public get 192(): string {
		return this.item(192) || undefined;
	}

	public get 193(): string {
		return this.item(193) || undefined;
	}

	public get 194(): string {
		return this.item(194) || undefined;
	}

	public get 195(): string {
		return this.item(195) || undefined;
	}

	public get 196(): string {
		return this.item(196) || undefined;
	}

	public get 197(): string {
		return this.item(197) || undefined;
	}

	public get 198(): string {
		return this.item(198) || undefined;
	}

	public get 199(): string {
		return this.item(199) || undefined;
	}

	public get 200(): string {
		return this.item(200) || undefined;
	}

	public get 201(): string {
		return this.item(201) || undefined;
	}

	public get 202(): string {
		return this.item(202) || undefined;
	}

	public get 203(): string {
		return this.item(203) || undefined;
	}

	public get 204(): string {
		return this.item(204) || undefined;
	}

	public get 205(): string {
		return this.item(205) || undefined;
	}

	public get 206(): string {
		return this.item(206) || undefined;
	}

	public get 207(): string {
		return this.item(207) || undefined;
	}

	public get 208(): string {
		return this.item(208) || undefined;
	}

	public get 209(): string {
		return this.item(209) || undefined;
	}

	public get 210(): string {
		return this.item(210) || undefined;
	}

	public get 211(): string {
		return this.item(211) || undefined;
	}

	public get 212(): string {
		return this.item(212) || undefined;
	}

	public get 213(): string {
		return this.item(213) || undefined;
	}

	public get 214(): string {
		return this.item(214) || undefined;
	}

	public get 215(): string {
		return this.item(215) || undefined;
	}

	public get 216(): string {
		return this.item(216) || undefined;
	}

	public get 217(): string {
		return this.item(217) || undefined;
	}

	public get 218(): string {
		return this.item(218) || undefined;
	}

	public get 219(): string {
		return this.item(219) || undefined;
	}

	public get 220(): string {
		return this.item(220) || undefined;
	}

	public get 221(): string {
		return this.item(221) || undefined;
	}

	public get 222(): string {
		return this.item(222) || undefined;
	}

	public get 223(): string {
		return this.item(223) || undefined;
	}

	public get 224(): string {
		return this.item(224) || undefined;
	}

	public get 225(): string {
		return this.item(225) || undefined;
	}

	public get 226(): string {
		return this.item(226) || undefined;
	}

	public get 227(): string {
		return this.item(227) || undefined;
	}

	public get 228(): string {
		return this.item(228) || undefined;
	}

	public get 229(): string {
		return this.item(229) || undefined;
	}

	public get 230(): string {
		return this.item(230) || undefined;
	}

	public get 231(): string {
		return this.item(231) || undefined;
	}

	public get 232(): string {
		return this.item(232) || undefined;
	}

	public get 233(): string {
		return this.item(233) || undefined;
	}

	public get 234(): string {
		return this.item(234) || undefined;
	}

	public get 235(): string {
		return this.item(235) || undefined;
	}

	public get 236(): string {
		return this.item(236) || undefined;
	}

	public get 237(): string {
		return this.item(237) || undefined;
	}

	public get 238(): string {
		return this.item(238) || undefined;
	}

	public get 239(): string {
		return this.item(239) || undefined;
	}

	public get 240(): string {
		return this.item(240) || undefined;
	}

	public get 241(): string {
		return this.item(241) || undefined;
	}

	public get 242(): string {
		return this.item(242) || undefined;
	}

	public get 243(): string {
		return this.item(243) || undefined;
	}

	public get 244(): string {
		return this.item(244) || undefined;
	}

	public get 245(): string {
		return this.item(245) || undefined;
	}

	public get 246(): string {
		return this.item(246) || undefined;
	}

	public get 247(): string {
		return this.item(247) || undefined;
	}

	public get 248(): string {
		return this.item(248) || undefined;
	}

	public get 249(): string {
		return this.item(249) || undefined;
	}

	public get 250(): string {
		return this.item(250) || undefined;
	}

	public get 251(): string {
		return this.item(251) || undefined;
	}

	public get 252(): string {
		return this.item(252) || undefined;
	}

	public get 253(): string {
		return this.item(253) || undefined;
	}

	public get 254(): string {
		return this.item(254) || undefined;
	}

	public get 255(): string {
		return this.item(255) || undefined;
	}

	public get 256(): string {
		return this.item(256) || undefined;
	}

	public get 257(): string {
		return this.item(257) || undefined;
	}

	public get 258(): string {
		return this.item(258) || undefined;
	}

	public get 259(): string {
		return this.item(259) || undefined;
	}

	public get 260(): string {
		return this.item(260) || undefined;
	}

	public get 261(): string {
		return this.item(261) || undefined;
	}

	public get 262(): string {
		return this.item(262) || undefined;
	}

	public get 263(): string {
		return this.item(263) || undefined;
	}

	public get 264(): string {
		return this.item(264) || undefined;
	}

	public get 265(): string {
		return this.item(265) || undefined;
	}

	public get 266(): string {
		return this.item(266) || undefined;
	}

	public get 267(): string {
		return this.item(267) || undefined;
	}

	public get 268(): string {
		return this.item(268) || undefined;
	}

	public get 269(): string {
		return this.item(269) || undefined;
	}

	public get 270(): string {
		return this.item(270) || undefined;
	}

	public get 271(): string {
		return this.item(271) || undefined;
	}

	public get 272(): string {
		return this.item(272) || undefined;
	}

	public get 273(): string {
		return this.item(273) || undefined;
	}

	public get 274(): string {
		return this.item(274) || undefined;
	}

	public get 275(): string {
		return this.item(275) || undefined;
	}

	public get 276(): string {
		return this.item(276) || undefined;
	}

	public get 277(): string {
		return this.item(277) || undefined;
	}

	public get 278(): string {
		return this.item(278) || undefined;
	}

	public get 279(): string {
		return this.item(279) || undefined;
	}

	public get 280(): string {
		return this.item(280) || undefined;
	}

	public get 281(): string {
		return this.item(281) || undefined;
	}

	public get 282(): string {
		return this.item(282) || undefined;
	}

	public get 283(): string {
		return this.item(283) || undefined;
	}

	public get 284(): string {
		return this.item(284) || undefined;
	}

	public get 285(): string {
		return this.item(285) || undefined;
	}

	public get 286(): string {
		return this.item(286) || undefined;
	}

	public get 287(): string {
		return this.item(287) || undefined;
	}

	public get 288(): string {
		return this.item(288) || undefined;
	}

	public get 289(): string {
		return this.item(289) || undefined;
	}

	public get 290(): string {
		return this.item(290) || undefined;
	}

	public get 291(): string {
		return this.item(291) || undefined;
	}

	public get 292(): string {
		return this.item(292) || undefined;
	}

	public get 293(): string {
		return this.item(293) || undefined;
	}

	public get 294(): string {
		return this.item(294) || undefined;
	}

	public get 295(): string {
		return this.item(295) || undefined;
	}

	public get 296(): string {
		return this.item(296) || undefined;
	}

	public get 297(): string {
		return this.item(297) || undefined;
	}

	public get 298(): string {
		return this.item(298) || undefined;
	}

	public get 299(): string {
		return this.item(299) || undefined;
	}

	public get 300(): string {
		return this.item(300) || undefined;
	}

	public get 301(): string {
		return this.item(301) || undefined;
	}

	public get 302(): string {
		return this.item(302) || undefined;
	}

	public get 303(): string {
		return this.item(303) || undefined;
	}

	public get 304(): string {
		return this.item(304) || undefined;
	}

	public get 305(): string {
		return this.item(305) || undefined;
	}

	public get 306(): string {
		return this.item(306) || undefined;
	}

	public get 307(): string {
		return this.item(307) || undefined;
	}

	public get 308(): string {
		return this.item(308) || undefined;
	}

	public get 309(): string {
		return this.item(309) || undefined;
	}

	public get 310(): string {
		return this.item(310) || undefined;
	}

	public get 311(): string {
		return this.item(311) || undefined;
	}

	public get 312(): string {
		return this.item(312) || undefined;
	}

	public get 313(): string {
		return this.item(313) || undefined;
	}

	public get 314(): string {
		return this.item(314) || undefined;
	}

	public get 315(): string {
		return this.item(315) || undefined;
	}

	public get 316(): string {
		return this.item(316) || undefined;
	}

	public get 317(): string {
		return this.item(317) || undefined;
	}

	public get 318(): string {
		return this.item(318) || undefined;
	}

	public get 319(): string {
		return this.item(319) || undefined;
	}

	public get 320(): string {
		return this.item(320) || undefined;
	}

	public get 321(): string {
		return this.item(321) || undefined;
	}

	public get 322(): string {
		return this.item(322) || undefined;
	}

	public get 323(): string {
		return this.item(323) || undefined;
	}

	public get 324(): string {
		return this.item(324) || undefined;
	}

	public get 325(): string {
		return this.item(325) || undefined;
	}

	public get 326(): string {
		return this.item(326) || undefined;
	}

	public get 327(): string {
		return this.item(327) || undefined;
	}

	public get 328(): string {
		return this.item(328) || undefined;
	}

	public get 329(): string {
		return this.item(329) || undefined;
	}

	public get 330(): string {
		return this.item(330) || undefined;
	}

	public get 331(): string {
		return this.item(331) || undefined;
	}

	public get 332(): string {
		return this.item(332) || undefined;
	}

	public get 333(): string {
		return this.item(333) || undefined;
	}

	public get 334(): string {
		return this.item(334) || undefined;
	}

	public get 335(): string {
		return this.item(335) || undefined;
	}

	public get 336(): string {
		return this.item(336) || undefined;
	}

	public get 337(): string {
		return this.item(337) || undefined;
	}

	public get 338(): string {
		return this.item(338) || undefined;
	}

	public get 339(): string {
		return this.item(339) || undefined;
	}

	public get 340(): string {
		return this.item(340) || undefined;
	}

	public get 341(): string {
		return this.item(341) || undefined;
	}

	public get 342(): string {
		return this.item(342) || undefined;
	}

	public get 343(): string {
		return this.item(343) || undefined;
	}

	public get 344(): string {
		return this.item(344) || undefined;
	}

	public get 345(): string {
		return this.item(345) || undefined;
	}

	public get 346(): string {
		return this.item(346) || undefined;
	}

	public get 347(): string {
		return this.item(347) || undefined;
	}

	public get 348(): string {
		return this.item(348) || undefined;
	}

	public get 349(): string {
		return this.item(349) || undefined;
	}

	public get 350(): string {
		return this.item(350) || undefined;
	}

	public get 351(): string {
		return this.item(351) || undefined;
	}

	public get 352(): string {
		return this.item(352) || undefined;
	}

	public get 353(): string {
		return this.item(353) || undefined;
	}

	public get 354(): string {
		return this.item(354) || undefined;
	}

	public get 355(): string {
		return this.item(355) || undefined;
	}

	public get 356(): string {
		return this.item(356) || undefined;
	}

	public get 357(): string {
		return this.item(357) || undefined;
	}

	public get 358(): string {
		return this.item(358) || undefined;
	}

	public get 359(): string {
		return this.item(359) || undefined;
	}

	public get 360(): string {
		return this.item(360) || undefined;
	}

	public get 361(): string {
		return this.item(361) || undefined;
	}

	public get 362(): string {
		return this.item(362) || undefined;
	}

	public get 363(): string {
		return this.item(363) || undefined;
	}

	public get 364(): string {
		return this.item(364) || undefined;
	}

	public get 365(): string {
		return this.item(365) || undefined;
	}

	public get 366(): string {
		return this.item(366) || undefined;
	}

	public get 367(): string {
		return this.item(367) || undefined;
	}

	public get 368(): string {
		return this.item(368) || undefined;
	}
	public get 369(): string {
		return this.item(369) || undefined;
	}

	public get 370(): string {
		return this.item(370) || undefined;
	}

	public get 371(): string {
		return this.item(371) || undefined;
	}

	public get 372(): string {
		return this.item(372) || undefined;
	}

	public get 373(): string {
		return this.item(373) || undefined;
	}

	public get 374(): string {
		return this.item(374) || undefined;
	}

	public get 375(): string {
		return this.item(375) || undefined;
	}

	public get 376(): string {
		return this.item(376) || undefined;
	}

	public get 377(): string {
		return this.item(377) || undefined;
	}

	public get 378(): string {
		return this.item(378) || undefined;
	}

	public get 379(): string {
		return this.item(379) || undefined;
	}

	public get 380(): string {
		return this.item(380) || undefined;
	}

	public get 381(): string {
		return this.item(381) || undefined;
	}

	public get 382(): string {
		return this.item(382) || undefined;
	}

	public get 383(): string {
		return this.item(383) || undefined;
	}

	public get 384(): string {
		return this.item(384) || undefined;
	}

	public get 385(): string {
		return this.item(385) || undefined;
	}

	public get 386(): string {
		return this.item(386) || undefined;
	}

	public get 387(): string {
		return this.item(387) || undefined;
	}

	public get 388(): string {
		return this.item(388) || undefined;
	}

	public get 389(): string {
		return this.item(389) || undefined;
	}

	public get 390(): string {
		return this.item(390) || undefined;
	}

	public get 391(): string {
		return this.item(391) || undefined;
	}

	public get 392(): string {
		return this.item(392) || undefined;
	}

	public get 393(): string {
		return this.item(393) || undefined;
	}

	/**
	 * CSS properties
	 */

	public get accentColor(): string {
		return this.getPropertyValue('accent-color');
	}

	public set accentColor(value: string) {
		this.setProperty('accent-color', value);
	}

	public get appRegion(): string {
		return this.getPropertyValue('app-region');
	}

	public set appRegion(value: string) {
		this.setProperty('app-region', value);
	}

	public get alignContent(): string {
		return this.getPropertyValue('align-content');
	}

	public set alignContent(value: string) {
		this.setProperty('align-content', value);
	}

	public get alignItems(): string {
		return this.getPropertyValue('align-items');
	}

	public set alignItems(value: string) {
		this.setProperty('align-items', value);
	}

	public get alignSelf(): string {
		return this.getPropertyValue('align-self');
	}

	public set alignSelf(value: string) {
		this.setProperty('align-self', value);
	}

	public get alignmentBaseline(): string {
		return this.getPropertyValue('alignment-baseline');
	}

	public set alignmentBaseline(value: string) {
		this.setProperty('alignment-baseline', value);
	}

	public get all(): string {
		return this.getPropertyValue('all');
	}

	public set all(value: string) {
		this.setProperty('all', value);
	}

	public get animation(): string {
		return this.getPropertyValue('animation');
	}

	public set animation(value: string) {
		this.setProperty('animation', value);
	}

	public get animationDelay(): string {
		return this.getPropertyValue('animation-delay');
	}

	public set animationDelay(value: string) {
		this.setProperty('animation-delay', value);
	}

	public get animationDirection(): string {
		return this.getPropertyValue('animation-direction');
	}

	public set animationDirection(value: string) {
		this.setProperty('animation-direction', value);
	}

	public get animationDuration(): string {
		return this.getPropertyValue('animation-duration');
	}

	public set animationDuration(value: string) {
		this.setProperty('animation-duration', value);
	}

	public get animationFillMode(): string {
		return this.getPropertyValue('animation-fill-mode');
	}

	public set animationFillMode(value: string) {
		this.setProperty('animation-fill-mode', value);
	}

	public get animationIterationCount(): string {
		return this.getPropertyValue('animation-iteration-count');
	}

	public set animationIterationCount(value: string) {
		this.setProperty('animation-iteration-count', value);
	}

	public get animationName(): string {
		return this.getPropertyValue('animation-name');
	}

	public set animationName(value: string) {
		this.setProperty('animation-name', value);
	}

	public get animationPlayState(): string {
		return this.getPropertyValue('animation-play-state');
	}

	public set animationPlayState(value: string) {
		this.setProperty('animation-play-state', value);
	}

	public get animationTimingFunction(): string {
		return this.getPropertyValue('animation-timing-function');
	}

	public set animationTimingFunction(value: string) {
		this.setProperty('animation-timing-function', value);
	}

	public get appearance(): string {
		return this.getPropertyValue('appearance');
	}

	public set appearance(value: string) {
		this.setProperty('appearance', value);
	}

	public get backdropFilter(): string {
		return this.getPropertyValue('backdrop-filter');
	}

	public set backdropFilter(value: string) {
		this.setProperty('backdrop-filter', value);
	}

	public get backfaceVisibility(): string {
		return this.getPropertyValue('backface-visibility');
	}

	public set backfaceVisibility(value: string) {
		this.setProperty('backface-visibility', value);
	}

	public get background(): string {
		return this.getPropertyValue('background');
	}

	public set background(value: string) {
		this.setProperty('background', value);
	}

	public get backgroundAttachment(): string {
		return this.getPropertyValue('background-attachment');
	}

	public set backgroundAttachment(value: string) {
		this.setProperty('background-attachment', value);
	}

	public get backgroundBlendMode(): string {
		return this.getPropertyValue('background-blend-mode');
	}

	public set backgroundBlendMode(value: string) {
		this.setProperty('background-blend-mode', value);
	}

	public get backgroundClip(): string {
		return this.getPropertyValue('background-clip');
	}

	public set backgroundClip(value: string) {
		this.setProperty('background-clip', value);
	}

	public get backgroundColor(): string {
		return this.getPropertyValue('background-color');
	}

	public set backgroundColor(value: string) {
		this.setProperty('background-color', value);
	}

	public get backgroundImage(): string {
		return this.getPropertyValue('background-image');
	}

	public set backgroundImage(value: string) {
		this.setProperty('background-image', value);
	}

	public get backgroundOrigin(): string {
		return this.getPropertyValue('background-origin');
	}

	public set backgroundOrigin(value: string) {
		this.setProperty('background-origin', value);
	}

	public get backgroundPosition(): string {
		return this.getPropertyValue('background-position');
	}

	public set backgroundPosition(value: string) {
		this.setProperty('background-position', value);
	}

	public get backgroundPositionX(): string {
		return this.getPropertyValue('background-position-x');
	}

	public set backgroundPositionX(value: string) {
		this.setProperty('background-position-x', value);
	}

	public get backgroundPositionY(): string {
		return this.getPropertyValue('background-position-y');
	}

	public set backgroundPositionY(value: string) {
		this.setProperty('background-position-y', value);
	}

	public get backgroundRepeat(): string {
		return this.getPropertyValue('background-repeat');
	}

	public set backgroundRepeat(value: string) {
		this.setProperty('background-repeat', value);
	}

	public get backgroundRepeatX(): string {
		return this.getPropertyValue('background-repeat-x');
	}

	public set backgroundRepeatX(value: string) {
		this.setProperty('background-repeat-x', value);
	}

	public get backgroundRepeatY(): string {
		return this.getPropertyValue('background-repeat-y');
	}

	public set backgroundRepeatY(value: string) {
		this.setProperty('background-repeat-y', value);
	}

	public get backgroundSize(): string {
		return this.getPropertyValue('background-size');
	}

	public set backgroundSize(value: string) {
		this.setProperty('background-size', value);
	}

	public get baselineShift(): string {
		return this.getPropertyValue('baseline-shift');
	}

	public set baselineShift(value: string) {
		this.setProperty('baseline-shift', value);
	}

	public get blockSize(): string {
		return this.getPropertyValue('block-size');
	}

	public set blockSize(value: string) {
		this.setProperty('block-size', value);
	}

	public get border(): string {
		return this.getPropertyValue('border');
	}

	public set border(value: string) {
		this.setProperty('border', value);
	}

	public get borderBlockEnd(): string {
		return this.getPropertyValue('border-block-end');
	}

	public set borderBlockEnd(value: string) {
		this.setProperty('border-block-end', value);
	}

	public get borderBlockEndColor(): string {
		return this.getPropertyValue('border-block-end-color');
	}

	public set borderBlockEndColor(value: string) {
		this.setProperty('border-block-end-color', value);
	}

	public get borderBlockEndStyle(): string {
		return this.getPropertyValue('border-block-end-style');
	}

	public set borderBlockEndStyle(value: string) {
		this.setProperty('border-block-end-style', value);
	}

	public get borderBlockEndWidth(): string {
		return this.getPropertyValue('border-block-end-width');
	}

	public set borderBlockEndWidth(value: string) {
		this.setProperty('border-block-end-width', value);
	}

	public get borderBlockStart(): string {
		return this.getPropertyValue('border-block-start');
	}

	public set borderBlockStart(value: string) {
		this.setProperty('border-block-start', value);
	}

	public get borderBlockStartColor(): string {
		return this.getPropertyValue('border-block-start-color');
	}

	public set borderBlockStartColor(value: string) {
		this.setProperty('border-block-start-color', value);
	}

	public get borderBlockStartStyle(): string {
		return this.getPropertyValue('border-block-start-style');
	}

	public set borderBlockStartStyle(value: string) {
		this.setProperty('border-block-start-style', value);
	}

	public get borderBlockStartWidth(): string {
		return this.getPropertyValue('border-block-start-width');
	}

	public set borderBlockStartWidth(value: string) {
		this.setProperty('border-block-start-width', value);
	}

	public get borderBottom(): string {
		return this.getPropertyValue('border-bottom');
	}

	public set borderBottom(value: string) {
		this.setProperty('border-bottom', value);
	}

	public get borderBottomColor(): string {
		return this.getPropertyValue('border-bottom-color');
	}

	public set borderBottomColor(value: string) {
		this.setProperty('border-bottom-color', value);
	}

	public get borderBottomLeftRadius(): string {
		return this.getPropertyValue('border-bottom-left-radius');
	}

	public set borderBottomLeftRadius(value: string) {
		this.setProperty('border-bottom-left-radius', value);
	}

	public get borderBottomRightRadius(): string {
		return this.getPropertyValue('border-bottom-right-radius');
	}

	public set borderBottomRightRadius(value: string) {
		this.setProperty('border-bottom-right-radius', value);
	}

	public get borderBottomStyle(): string {
		return this.getPropertyValue('border-bottom-style');
	}

	public set borderBottomStyle(value: string) {
		this.setProperty('border-bottom-style', value);
	}

	public get borderBottomWidth(): string {
		return this.getPropertyValue('border-bottom-width');
	}

	public set borderBottomWidth(value: string) {
		this.setProperty('border-bottom-width', value);
	}

	public get borderCollapse(): string {
		return this.getPropertyValue('border-collapse');
	}

	public set borderCollapse(value: string) {
		this.setProperty('border-collapse', value);
	}

	public get borderColor(): string {
		return this.getPropertyValue('border-color');
	}

	public set borderColor(value: string) {
		this.setProperty('border-color', value);
	}

	public get borderImage(): string {
		return this.getPropertyValue('border-image');
	}

	public set borderImage(value: string) {
		this.setProperty('border-image', value);
	}

	public get borderImageOutset(): string {
		return this.getPropertyValue('border-image-outset');
	}

	public set borderImageOutset(value: string) {
		this.setProperty('border-image-outset', value);
	}

	public get borderImageRepeat(): string {
		return this.getPropertyValue('border-image-repeat');
	}

	public set borderImageRepeat(value: string) {
		this.setProperty('border-image-repeat', value);
	}

	public get borderImageSlice(): string {
		return this.getPropertyValue('border-image-slice');
	}

	public set borderImageSlice(value: string) {
		this.setProperty('border-image-slice', value);
	}

	public get borderImageSource(): string {
		return this.getPropertyValue('border-image-source');
	}

	public set borderImageSource(value: string) {
		this.setProperty('border-image-source', value);
	}

	public get borderImageWidth(): string {
		return this.getPropertyValue('border-image-width');
	}

	public set borderImageWidth(value: string) {
		this.setProperty('border-image-width', value);
	}

	public get borderInlineEnd(): string {
		return this.getPropertyValue('border-inline-end');
	}

	public set borderInlineEnd(value: string) {
		this.setProperty('border-inline-end', value);
	}

	public get borderInlineEndColor(): string {
		return this.getPropertyValue('border-inline-end-color');
	}

	public set borderInlineEndColor(value: string) {
		this.setProperty('border-inline-end-color', value);
	}

	public get borderInlineEndStyle(): string {
		return this.getPropertyValue('border-inline-end-style');
	}

	public set borderInlineEndStyle(value: string) {
		this.setProperty('border-inline-end-style', value);
	}

	public get borderInlineEndWidth(): string {
		return this.getPropertyValue('border-inline-end-width');
	}

	public set borderInlineEndWidth(value: string) {
		this.setProperty('border-inline-end-width', value);
	}

	public get borderInlineStart(): string {
		return this.getPropertyValue('border-inline-start');
	}

	public set borderInlineStart(value: string) {
		this.setProperty('border-inline-start', value);
	}

	public get borderInlineStartColor(): string {
		return this.getPropertyValue('border-inline-start-color');
	}

	public set borderInlineStartColor(value: string) {
		this.setProperty('border-inline-start-color', value);
	}

	public get borderInlineStartStyle(): string {
		return this.getPropertyValue('border-inline-start-style');
	}

	public set borderInlineStartStyle(value: string) {
		this.setProperty('border-inline-start-style', value);
	}

	public get borderInlineStartWidth(): string {
		return this.getPropertyValue('border-inline-start-width');
	}

	public set borderInlineStartWidth(value: string) {
		this.setProperty('border-inline-start-width', value);
	}

	public get borderLeft(): string {
		return this.getPropertyValue('border-left');
	}

	public set borderLeft(value: string) {
		this.setProperty('border-left', value);
	}

	public get borderLeftColor(): string {
		return this.getPropertyValue('border-left-color');
	}

	public set borderLeftColor(value: string) {
		this.setProperty('border-left-color', value);
	}

	public get borderLeftStyle(): string {
		return this.getPropertyValue('border-left-style');
	}

	public set borderLeftStyle(value: string) {
		this.setProperty('border-left-style', value);
	}

	public get borderLeftWidth(): string {
		return this.getPropertyValue('border-left-width');
	}

	public set borderLeftWidth(value: string) {
		this.setProperty('border-left-width', value);
	}

	public get borderRadius(): string {
		return this.getPropertyValue('border-radius');
	}

	public set borderRadius(value: string) {
		this.setProperty('border-radius', value);
	}

	public get borderRight(): string {
		return this.getPropertyValue('border-right');
	}

	public set borderRight(value: string) {
		this.setProperty('border-right', value);
	}

	public get borderRightColor(): string {
		return this.getPropertyValue('border-right-color');
	}

	public set borderRightColor(value: string) {
		this.setProperty('border-right-color', value);
	}

	public get borderRightStyle(): string {
		return this.getPropertyValue('border-right-style');
	}

	public set borderRightStyle(value: string) {
		this.setProperty('border-right-style', value);
	}

	public get borderRightWidth(): string {
		return this.getPropertyValue('border-right-width');
	}

	public set borderRightWidth(value: string) {
		this.setProperty('border-right-width', value);
	}

	public get borderSpacing(): string {
		return this.getPropertyValue('border-spacing');
	}

	public set borderSpacing(value: string) {
		this.setProperty('border-spacing', value);
	}

	public get borderStyle(): string {
		return this.getPropertyValue('border-style');
	}

	public set borderStyle(value: string) {
		this.setProperty('border-style', value);
	}

	public get borderTop(): string {
		return this.getPropertyValue('border-top');
	}

	public set borderTop(value: string) {
		this.setProperty('border-top', value);
	}

	public get borderTopColor(): string {
		return this.getPropertyValue('border-top-color');
	}

	public set borderTopColor(value: string) {
		this.setProperty('border-top-color', value);
	}

	public get borderTopLeftRadius(): string {
		return this.getPropertyValue('border-top-left-radius');
	}

	public set borderTopLeftRadius(value: string) {
		this.setProperty('border-top-left-radius', value);
	}

	public get borderTopRightRadius(): string {
		return this.getPropertyValue('border-top-right-radius');
	}

	public set borderTopRightRadius(value: string) {
		this.setProperty('border-top-right-radius', value);
	}

	public get borderTopStyle(): string {
		return this.getPropertyValue('border-top-style');
	}

	public set borderTopStyle(value: string) {
		this.setProperty('border-top-style', value);
	}

	public get borderTopWidth(): string {
		return this.getPropertyValue('border-top-width');
	}

	public set borderTopWidth(value: string) {
		this.setProperty('border-top-width', value);
	}

	public get borderWidth(): string {
		return this.getPropertyValue('border-width');
	}

	public set borderWidth(value: string) {
		this.setProperty('border-width', value);
	}

	public get borderEndEndRadius(): string {
		return this.getPropertyValue('border-end-end-radius');
	}

	public set borderEndEndRadius(value: string) {
		this.setProperty('border-end-end-radius', value);
	}

	public get borderEndStartRadius(): string {
		return this.getPropertyValue('border-end-start-radius');
	}

	public set borderEndStartRadius(value: string) {
		this.setProperty('border-end-start-radius', value);
	}

	public get borderStartEndRadius(): string {
		return this.getPropertyValue('border-start-end-radius');
	}

	public set borderStartEndRadius(value: string) {
		this.setProperty('border-start-end-radius', value);
	}

	public get borderStartStartRadius(): string {
		return this.getPropertyValue('border-start-start-radius');
	}

	public set borderStartStartRadius(value: string) {
		this.setProperty('border-start-start-radius', value);
	}

	public get bottom(): string {
		return this.getPropertyValue('bottom');
	}

	public set bottom(value: string) {
		this.setProperty('bottom', value);
	}

	public get boxShadow(): string {
		return this.getPropertyValue('box-shadow');
	}

	public set boxShadow(value: string) {
		this.setProperty('box-shadow', value);
	}

	public get boxSizing(): string {
		return this.getPropertyValue('box-sizing');
	}

	public set boxSizing(value: string) {
		this.setProperty('box-sizing', value);
	}

	public get breakAfter(): string {
		return this.getPropertyValue('break-after');
	}

	public set breakAfter(value: string) {
		this.setProperty('break-after', value);
	}

	public get breakBefore(): string {
		return this.getPropertyValue('break-before');
	}

	public set breakBefore(value: string) {
		this.setProperty('break-before', value);
	}

	public get breakInside(): string {
		return this.getPropertyValue('break-inside');
	}

	public set breakInside(value: string) {
		this.setProperty('break-inside', value);
	}

	public get bufferedRendering(): string {
		return this.getPropertyValue('buffered-rendering');
	}

	public set bufferedRendering(value: string) {
		this.setProperty('buffered-rendering', value);
	}

	public get captionSide(): string {
		return this.getPropertyValue('caption-side');
	}

	public set captionSide(value: string) {
		this.setProperty('caption-side', value);
	}

	public get caretColor(): string {
		return this.getPropertyValue('caret-color');
	}

	public set caretColor(value: string) {
		this.setProperty('caret-color', value);
	}

	public get clear(): string {
		return this.getPropertyValue('clear');
	}

	public set clear(value: string) {
		this.setProperty('clear', value);
	}

	public get clip(): string {
		return this.getPropertyValue('clip');
	}

	public set clip(value: string) {
		this.setProperty('clip', value);
	}

	public get clipPath(): string {
		return this.getPropertyValue('clip-path');
	}

	public set clipPath(value: string) {
		this.setProperty('clip-path', value);
	}

	public get clipRule(): string {
		return this.getPropertyValue('clip-rule');
	}

	public set clipRule(value: string) {
		this.setProperty('clip-rule', value);
	}

	public get color(): string {
		return this.getPropertyValue('color');
	}

	public set color(value: string) {
		this.setProperty('color', value);
	}

	public get colorInterpolation(): string {
		return this.getPropertyValue('color-interpolation');
	}

	public set colorInterpolation(value: string) {
		this.setProperty('color-interpolation', value);
	}

	public get colorInterpolationFilters(): string {
		return this.getPropertyValue('color-interpolation-filters');
	}

	public set colorInterpolationFilters(value: string) {
		this.setProperty('color-interpolation-filters', value);
	}

	public get colorRendering(): string {
		return this.getPropertyValue('color-rendering');
	}

	public set colorRendering(value: string) {
		this.setProperty('color-rendering', value);
	}

	public get colorScheme(): string {
		return this.getPropertyValue('color-scheme');
	}

	public set colorScheme(value: string) {
		this.setProperty('color-scheme', value);
	}

	public get columnCount(): string {
		return this.getPropertyValue('column-count');
	}

	public set columnCount(value: string) {
		this.setProperty('column-count', value);
	}

	public get columnFill(): string {
		return this.getPropertyValue('column-fill');
	}

	public set columnFill(value: string) {
		this.setProperty('column-fill', value);
	}

	public get columnGap(): string {
		return this.getPropertyValue('column-gap');
	}

	public set columnGap(value: string) {
		this.setProperty('column-gap', value);
	}

	public get columnRule(): string {
		return this.getPropertyValue('column-rule');
	}

	public set columnRule(value: string) {
		this.setProperty('column-rule', value);
	}

	public get columnRuleColor(): string {
		return this.getPropertyValue('column-rule-color');
	}

	public set columnRuleColor(value: string) {
		this.setProperty('column-rule-color', value);
	}

	public get columnRuleStyle(): string {
		return this.getPropertyValue('column-rule-style');
	}

	public set columnRuleStyle(value: string) {
		this.setProperty('column-rule-style', value);
	}

	public get columnRuleWidth(): string {
		return this.getPropertyValue('column-rule-width');
	}

	public set columnRuleWidth(value: string) {
		this.setProperty('column-rule-width', value);
	}

	public get columnSpan(): string {
		return this.getPropertyValue('column-span');
	}

	public set columnSpan(value: string) {
		this.setProperty('column-span', value);
	}

	public get columnWidth(): string {
		return this.getPropertyValue('column-width');
	}

	public set columnWidth(value: string) {
		this.setProperty('column-width', value);
	}

	public get columns(): string {
		return this.getPropertyValue('columns');
	}

	public set columns(value: string) {
		this.setProperty('columns', value);
	}

	public get contain(): string {
		return this.getPropertyValue('contain');
	}

	public set contain(value: string) {
		this.setProperty('contain', value);
	}

	public get containIntrinsicSize(): string {
		return this.getPropertyValue('contain-intrinsic-size');
	}

	public set containIntrinsicSize(value: string) {
		this.setProperty('contain-intrinsic-size', value);
	}

	public get content(): string {
		return this.getPropertyValue('content');
	}

	public set content(value: string) {
		this.setProperty('content', value);
	}

	public get contentVisibility(): string {
		return this.getPropertyValue('content-visibility');
	}

	public set contentVisibility(value: string) {
		this.setProperty('content-visibility', value);
	}

	public get counterIncrement(): string {
		return this.getPropertyValue('counter-increment');
	}

	public set counterIncrement(value: string) {
		this.setProperty('counter-increment', value);
	}

	public get counterReset(): string {
		return this.getPropertyValue('counter-reset');
	}

	public set counterReset(value: string) {
		this.setProperty('counter-reset', value);
	}

	public get counterSet(): string {
		return this.getPropertyValue('counter-set');
	}

	public set counterSet(value: string) {
		this.setProperty('counter-set', value);
	}

	public get containIntrinsicBlockSize(): string {
		return this.getPropertyValue('contain-intrinsic-block-size');
	}

	public set containIntrinsicBlockSize(value: string) {
		this.setProperty('contain-intrinsic-block-size', value);
	}

	public get containIntrinsicHeight(): string {
		return this.getPropertyValue('contain-intrinsic-height');
	}

	public set containIntrinsicHeight(value: string) {
		this.setProperty('contain-intrinsic-height', value);
	}

	public get containIntrinsicInlineSize(): string {
		return this.getPropertyValue('contain-intrinsic-inline-size');
	}

	public set containIntrinsicInlineSize(value: string) {
		this.setProperty('contain-intrinsic-inline-size', value);
	}

	public get containIntrinsicWidth(): string {
		return this.getPropertyValue('contain-intrinsic-width');
	}

	public set containIntrinsicWidth(value: string) {
		this.setProperty('contain-intrinsic-width', value);
	}

	public get cssFloat(): string {
		return this.getPropertyValue('css-float');
	}

	public set cssFloat(value: string) {
		this.setProperty('css-float', value);
	}

	public get cursor(): string {
		return this.getPropertyValue('cursor');
	}

	public set cursor(value: string) {
		this.setProperty('cursor', value);
	}

	public get cx(): string {
		return this.getPropertyValue('cx');
	}

	public set cx(value: string) {
		this.setProperty('cx', value);
	}

	public get cy(): string {
		return this.getPropertyValue('cy');
	}

	public set cy(value: string) {
		this.setProperty('cy', value);
	}

	public get d(): string {
		return this.getPropertyValue('d');
	}

	public set d(value: string) {
		this.setProperty('d', value);
	}

	public get direction(): string {
		return this.getPropertyValue('direction');
	}

	public set direction(value: string) {
		this.setProperty('direction', value);
	}

	public get display(): string {
		return this.getPropertyValue('display');
	}

	public set display(value: string) {
		this.setProperty('display', value);
	}

	public get dominantBaseline(): string {
		return this.getPropertyValue('dominant-baseline');
	}

	public set dominantBaseline(value: string) {
		this.setProperty('dominant-baseline', value);
	}

	public get emptyCells(): string {
		return this.getPropertyValue('empty-cells');
	}

	public set emptyCells(value: string) {
		this.setProperty('empty-cells', value);
	}

	public get fill(): string {
		return this.getPropertyValue('fill');
	}

	public set fill(value: string) {
		this.setProperty('fill', value);
	}

	public get fillOpacity(): string {
		return this.getPropertyValue('fill-opacity');
	}

	public set fillOpacity(value: string) {
		this.setProperty('fill-opacity', value);
	}

	public get fillRule(): string {
		return this.getPropertyValue('fill-rule');
	}

	public set fillRule(value: string) {
		this.setProperty('fill-rule', value);
	}

	public get filter(): string {
		return this.getPropertyValue('filter');
	}

	public set filter(value: string) {
		this.setProperty('filter', value);
	}

	public get flex(): string {
		return this.getPropertyValue('flex');
	}

	public set flex(value: string) {
		this.setProperty('flex', value);
	}

	public get flexBasis(): string {
		return this.getPropertyValue('flex-basis');
	}

	public set flexBasis(value: string) {
		this.setProperty('flex-basis', value);
	}

	public get flexDirection(): string {
		return this.getPropertyValue('flex-direction');
	}

	public set flexDirection(value: string) {
		this.setProperty('flex-direction', value);
	}

	public get flexFlow(): string {
		return this.getPropertyValue('flex-flow');
	}

	public set flexFlow(value: string) {
		this.setProperty('flex-flow', value);
	}

	public get flexGrow(): string {
		return this.getPropertyValue('flex-grow');
	}

	public set flexGrow(value: string) {
		this.setProperty('flex-grow', value);
	}

	public get flexShrink(): string {
		return this.getPropertyValue('flex-shrink');
	}

	public set flexShrink(value: string) {
		this.setProperty('flex-shrink', value);
	}

	public get flexWrap(): string {
		return this.getPropertyValue('flex-wrap');
	}

	public set flexWrap(value: string) {
		this.setProperty('flex-wrap', value);
	}

	public get float(): string {
		return this.getPropertyValue('float');
	}

	public set float(value: string) {
		this.setProperty('float', value);
	}

	public get floodColor(): string {
		return this.getPropertyValue('flood-color');
	}

	public set floodColor(value: string) {
		this.setProperty('flood-color', value);
	}

	public get floodOpacity(): string {
		return this.getPropertyValue('flood-opacity');
	}

	public set floodOpacity(value: string) {
		this.setProperty('flood-opacity', value);
	}

	public get font(): string {
		return this.getPropertyValue('font');
	}

	public set font(value: string) {
		this.setProperty('font', value);
	}

	public get fontDisplay(): string {
		return this.getPropertyValue('font-display');
	}

	public set fontDisplay(value: string) {
		this.setProperty('font-display', value);
	}

	public get fontFamily(): string {
		return this.getPropertyValue('font-family');
	}

	public set fontFamily(value: string) {
		this.setProperty('font-family', value);
	}

	public get fontFeatureSettings(): string {
		return this.getPropertyValue('font-feature-settings');
	}

	public set fontFeatureSettings(value: string) {
		this.setProperty('font-feature-settings', value);
	}

	public get fontKerning(): string {
		return this.getPropertyValue('font-kerning');
	}

	public set fontKerning(value: string) {
		this.setProperty('font-kerning', value);
	}

	public get fontOpticalSizing(): string {
		return this.getPropertyValue('font-optical-sizing');
	}

	public set fontOpticalSizing(value: string) {
		this.setProperty('font-optical-sizing', value);
	}

	public get fontSize(): string {
		return this.getPropertyValue('font-size');
	}

	public set fontSize(value: string) {
		this.setProperty('font-size', value);
	}

	public get fontStretch(): string {
		return this.getPropertyValue('font-stretch');
	}

	public set fontStretch(value: string) {
		this.setProperty('font-stretch', value);
	}

	public get fontStyle(): string {
		return this.getPropertyValue('font-style');
	}

	public set fontStyle(value: string) {
		this.setProperty('font-style', value);
	}

	public get fontVariant(): string {
		return this.getPropertyValue('font-variant');
	}

	public set fontVariant(value: string) {
		this.setProperty('font-variant', value);
	}

	public get fontVariantCaps(): string {
		return this.getPropertyValue('font-variant-caps');
	}

	public set fontVariantCaps(value: string) {
		this.setProperty('font-variant-caps', value);
	}

	public get fontVariantEastAsian(): string {
		return this.getPropertyValue('font-variant-east-asian');
	}

	public set fontVariantEastAsian(value: string) {
		this.setProperty('font-variant-east-asian', value);
	}

	public get fontVariantLigatures(): string {
		return this.getPropertyValue('font-variant-ligatures');
	}

	public set fontVariantLigatures(value: string) {
		this.setProperty('font-variant-ligatures', value);
	}

	public get fontVariantNumeric(): string {
		return this.getPropertyValue('font-variant-numeric');
	}

	public set fontVariantNumeric(value: string) {
		this.setProperty('font-variant-numeric', value);
	}

	public get fontVariationSettings(): string {
		return this.getPropertyValue('font-variation-settings');
	}

	public set fontVariationSettings(value: string) {
		this.setProperty('font-variation-settings', value);
	}

	public get fontPalette(): string {
		return this.getPropertyValue('font-palette');
	}

	public set fontPalette(value: string) {
		this.setProperty('font-palette', value);
	}

	public get fontSynthesisSmallCaps(): string {
		return this.getPropertyValue('font-synthesis-small-caps');
	}

	public set fontSynthesisSmallCaps(value: string) {
		this.setProperty('font-synthesis-small-caps', value);
	}

	public get fontSynthesisStyle(): string {
		return this.getPropertyValue('font-synthesis-style');
	}

	public set fontSynthesisStyle(value: string) {
		this.setProperty('font-synthesis-style', value);
	}

	public get fontSynthesisWeight(): string {
		return this.getPropertyValue('font-synthesis-weight');
	}

	public set fontSynthesisWeight(value: string) {
		this.setProperty('font-synthesis-weight', value);
	}

	public get fontWeight(): string {
		return this.getPropertyValue('font-weight');
	}

	public set fontWeight(value: string) {
		this.setProperty('font-weight', value);
	}

	public get gap(): string {
		return this.getPropertyValue('gap');
	}

	public set gap(value: string) {
		this.setProperty('gap', value);
	}

	public get grid(): string {
		return this.getPropertyValue('grid');
	}

	public set grid(value: string) {
		this.setProperty('grid', value);
	}

	public get gridArea(): string {
		return this.getPropertyValue('grid-area');
	}

	public set gridArea(value: string) {
		this.setProperty('grid-area', value);
	}

	public get gridAutoColumns(): string {
		return this.getPropertyValue('grid-auto-columns');
	}

	public set gridAutoColumns(value: string) {
		this.setProperty('grid-auto-columns', value);
	}

	public get gridAutoFlow(): string {
		return this.getPropertyValue('grid-auto-flow');
	}

	public set gridAutoFlow(value: string) {
		this.setProperty('grid-auto-flow', value);
	}

	public get gridAutoRows(): string {
		return this.getPropertyValue('grid-auto-rows');
	}

	public set gridAutoRows(value: string) {
		this.setProperty('grid-auto-rows', value);
	}

	public get gridColumn(): string {
		return this.getPropertyValue('grid-column');
	}

	public set gridColumn(value: string) {
		this.setProperty('grid-column', value);
	}

	public get gridColumnEnd(): string {
		return this.getPropertyValue('grid-column-end');
	}

	public set gridColumnEnd(value: string) {
		this.setProperty('grid-column-end', value);
	}

	public get gridColumnGap(): string {
		return this.getPropertyValue('grid-column-gap');
	}

	public set gridColumnGap(value: string) {
		this.setProperty('grid-column-gap', value);
	}

	public get gridColumnStart(): string {
		return this.getPropertyValue('grid-column-start');
	}

	public set gridColumnStart(value: string) {
		this.setProperty('grid-column-start', value);
	}

	public get gridGap(): string {
		return this.getPropertyValue('grid-gap');
	}

	public set gridGap(value: string) {
		this.setProperty('grid-gap', value);
	}

	public get gridRow(): string {
		return this.getPropertyValue('grid-row');
	}

	public set gridRow(value: string) {
		this.setProperty('grid-row', value);
	}

	public get gridRowEnd(): string {
		return this.getPropertyValue('grid-row-end');
	}

	public set gridRowEnd(value: string) {
		this.setProperty('grid-row-end', value);
	}

	public get gridRowGap(): string {
		return this.getPropertyValue('grid-row-gap');
	}

	public set gridRowGap(value: string) {
		this.setProperty('grid-row-gap', value);
	}

	public get gridRowStart(): string {
		return this.getPropertyValue('grid-row-start');
	}

	public set gridRowStart(value: string) {
		this.setProperty('grid-row-start', value);
	}

	public get gridTemplate(): string {
		return this.getPropertyValue('grid-template');
	}

	public set gridTemplate(value: string) {
		this.setProperty('grid-template', value);
	}

	public get gridTemplateAreas(): string {
		return this.getPropertyValue('grid-template-areas');
	}

	public set gridTemplateAreas(value: string) {
		this.setProperty('grid-template-areas', value);
	}

	public get gridTemplateColumns(): string {
		return this.getPropertyValue('grid-template-columns');
	}

	public set gridTemplateColumns(value: string) {
		this.setProperty('grid-template-columns', value);
	}

	public get gridTemplateRows(): string {
		return this.getPropertyValue('grid-template-rows');
	}

	public set gridTemplateRows(value: string) {
		this.setProperty('grid-template-rows', value);
	}

	public get height(): string {
		return this.getPropertyValue('height');
	}

	public set height(value: string) {
		this.setProperty('height', value);
	}

	public get hyphens(): string {
		return this.getPropertyValue('hyphens');
	}

	public set hyphens(value: string) {
		this.setProperty('hyphens', value);
	}

	public get imageOrientation(): string {
		return this.getPropertyValue('image-orientation');
	}

	public set imageOrientation(value: string) {
		this.setProperty('image-orientation', value);
	}

	public get imageRendering(): string {
		return this.getPropertyValue('image-rendering');
	}

	public set imageRendering(value: string) {
		this.setProperty('image-rendering', value);
	}

	public get inherits(): string {
		return this.getPropertyValue('inherits');
	}

	public set inherits(value: string) {
		this.setProperty('inherits', value);
	}

	public get initialValue(): string {
		return this.getPropertyValue('initial-value');
	}

	public set initialValue(value: string) {
		this.setProperty('initial-value', value);
	}

	public get inlineSize(): string {
		return this.getPropertyValue('inline-size');
	}

	public set inlineSize(value: string) {
		this.setProperty('inline-size', value);
	}

	public get isolation(): string {
		return this.getPropertyValue('isolation');
	}

	public set isolation(value: string) {
		this.setProperty('isolation', value);
	}

	public get insetBlockEnd(): string {
		return this.getPropertyValue('inset-block-end');
	}

	public set insetBlockEnd(value: string) {
		this.setProperty('inset-block-end', value);
	}

	public get insetBlockStart(): string {
		return this.getPropertyValue('inset-block-start');
	}

	public set insetBlockStart(value: string) {
		this.setProperty('inset-block-start', value);
	}

	public get insetInlineEnd(): string {
		return this.getPropertyValue('inset-inline-end');
	}

	public set insetInlineEnd(value: string) {
		this.setProperty('inset-inline-end', value);
	}

	public get insetInlineStart(): string {
		return this.getPropertyValue('inset-inline-start');
	}

	public set insetInlineStart(value: string) {
		this.setProperty('inset-inline-start', value);
	}

	public get justifyContent(): string {
		return this.getPropertyValue('justify-content');
	}

	public set justifyContent(value: string) {
		this.setProperty('justify-content', value);
	}

	public get justifyItems(): string {
		return this.getPropertyValue('justify-items');
	}

	public set justifyItems(value: string) {
		this.setProperty('justify-items', value);
	}

	public get justifySelf(): string {
		return this.getPropertyValue('justify-self');
	}

	public set justifySelf(value: string) {
		this.setProperty('justify-self', value);
	}

	public get left(): string {
		return this.getPropertyValue('left');
	}

	public set left(value: string) {
		this.setProperty('left', value);
	}

	public get letterSpacing(): string {
		return this.getPropertyValue('letter-spacing');
	}

	public set letterSpacing(value: string) {
		this.setProperty('letter-spacing', value);
	}

	public get lightingColor(): string {
		return this.getPropertyValue('lighting-color');
	}

	public set lightingColor(value: string) {
		this.setProperty('lighting-color', value);
	}

	public get lineBreak(): string {
		return this.getPropertyValue('line-break');
	}

	public set lineBreak(value: string) {
		this.setProperty('line-break', value);
	}

	public get lineHeight(): string {
		return this.getPropertyValue('line-height');
	}

	public set lineHeight(value: string) {
		this.setProperty('line-height', value);
	}

	public get listStyle(): string {
		return this.getPropertyValue('list-style');
	}

	public set listStyle(value: string) {
		this.setProperty('list-style', value);
	}

	public get listStyleImage(): string {
		return this.getPropertyValue('list-style-image');
	}

	public set listStyleImage(value: string) {
		this.setProperty('list-style-image', value);
	}

	public get listStylePosition(): string {
		return this.getPropertyValue('list-style-position');
	}

	public set listStylePosition(value: string) {
		this.setProperty('list-style-position', value);
	}

	public get listStyleType(): string {
		return this.getPropertyValue('list-style-type');
	}

	public set listStyleType(value: string) {
		this.setProperty('list-style-type', value);
	}

	public get margin(): string {
		return this.getPropertyValue('margin');
	}

	public set margin(value: string) {
		this.setProperty('margin', value);
	}

	public get marginBlockEnd(): string {
		return this.getPropertyValue('margin-block-end');
	}

	public set marginBlockEnd(value: string) {
		this.setProperty('margin-block-end', value);
	}

	public get marginBlockStart(): string {
		return this.getPropertyValue('margin-block-start');
	}

	public set marginBlockStart(value: string) {
		this.setProperty('margin-block-start', value);
	}

	public get marginBottom(): string {
		return this.getPropertyValue('margin-bottom');
	}

	public set marginBottom(value: string) {
		this.setProperty('margin-bottom', value);
	}

	public get marginInlineEnd(): string {
		return this.getPropertyValue('margin-inline-end');
	}

	public set marginInlineEnd(value: string) {
		this.setProperty('margin-inline-end', value);
	}

	public get marginInlineStart(): string {
		return this.getPropertyValue('margin-inline-start');
	}

	public set marginInlineStart(value: string) {
		this.setProperty('margin-inline-start', value);
	}

	public get marginLeft(): string {
		return this.getPropertyValue('margin-left');
	}

	public set marginLeft(value: string) {
		this.setProperty('margin-left', value);
	}

	public get marginRight(): string {
		return this.getPropertyValue('margin-right');
	}

	public set marginRight(value: string) {
		this.setProperty('margin-right', value);
	}

	public get marginTop(): string {
		return this.getPropertyValue('margin-top');
	}

	public set marginTop(value: string) {
		this.setProperty('margin-top', value);
	}

	public get marker(): string {
		return this.getPropertyValue('marker');
	}

	public set marker(value: string) {
		this.setProperty('marker', value);
	}

	public get markerEnd(): string {
		return this.getPropertyValue('marker-end');
	}

	public set markerEnd(value: string) {
		this.setProperty('marker-end', value);
	}

	public get markerMid(): string {
		return this.getPropertyValue('marker-mid');
	}

	public set markerMid(value: string) {
		this.setProperty('marker-mid', value);
	}

	public get markerStart(): string {
		return this.getPropertyValue('marker-start');
	}

	public set markerStart(value: string) {
		this.setProperty('marker-start', value);
	}

	public get mask(): string {
		return this.getPropertyValue('mask');
	}

	public set mask(value: string) {
		this.setProperty('mask', value);
	}

	public get maskType(): string {
		return this.getPropertyValue('mask-type');
	}

	public set maskType(value: string) {
		this.setProperty('mask-type', value);
	}

	public get maxBlockSize(): string {
		return this.getPropertyValue('max-block-size');
	}

	public set maxBlockSize(value: string) {
		this.setProperty('max-block-size', value);
	}

	public get maxHeight(): string {
		return this.getPropertyValue('max-height');
	}

	public set maxHeight(value: string) {
		this.setProperty('max-height', value);
	}

	public get maxInlineSize(): string {
		return this.getPropertyValue('max-inline-size');
	}

	public set maxInlineSize(value: string) {
		this.setProperty('max-inline-size', value);
	}

	public get maxWidth(): string {
		return this.getPropertyValue('max-width');
	}

	public set maxWidth(value: string) {
		this.setProperty('max-width', value);
	}

	public get maxZoom(): string {
		return this.getPropertyValue('max-zoom');
	}

	public set maxZoom(value: string) {
		this.setProperty('max-zoom', value);
	}

	public get minBlockSize(): string {
		return this.getPropertyValue('min-block-size');
	}

	public set minBlockSize(value: string) {
		this.setProperty('min-block-size', value);
	}

	public get minHeight(): string {
		return this.getPropertyValue('min-height');
	}

	public set minHeight(value: string) {
		this.setProperty('min-height', value);
	}

	public get minInlineSize(): string {
		return this.getPropertyValue('min-inline-size');
	}

	public set minInlineSize(value: string) {
		this.setProperty('min-inline-size', value);
	}

	public get minWidth(): string {
		return this.getPropertyValue('min-width');
	}

	public set minWidth(value: string) {
		this.setProperty('min-width', value);
	}

	public get minZoom(): string {
		return this.getPropertyValue('min-zoom');
	}

	public set minZoom(value: string) {
		this.setProperty('min-zoom', value);
	}

	public get mixBlendMode(): string {
		return this.getPropertyValue('mix-blend-mode');
	}

	public set mixBlendMode(value: string) {
		this.setProperty('mix-blend-mode', value);
	}

	public get objectFit(): string {
		return this.getPropertyValue('object-fit');
	}

	public set objectFit(value: string) {
		this.setProperty('object-fit', value);
	}

	public get objectPosition(): string {
		return this.getPropertyValue('object-position');
	}

	public set objectPosition(value: string) {
		this.setProperty('object-position', value);
	}

	public get offset(): string {
		return this.getPropertyValue('offset');
	}

	public set offset(value: string) {
		this.setProperty('offset', value);
	}

	public get offsetDistance(): string {
		return this.getPropertyValue('offset-distance');
	}

	public set offsetDistance(value: string) {
		this.setProperty('offset-distance', value);
	}

	public get offsetPath(): string {
		return this.getPropertyValue('offset-path');
	}

	public set offsetPath(value: string) {
		this.setProperty('offset-path', value);
	}

	public get offsetRotate(): string {
		return this.getPropertyValue('offset-rotate');
	}

	public set offsetRotate(value: string) {
		this.setProperty('offset-rotate', value);
	}

	public get opacity(): string {
		return this.getPropertyValue('opacity');
	}

	public set opacity(value: string) {
		this.setProperty('opacity', value);
	}

	public get order(): string {
		return this.getPropertyValue('order');
	}

	public set order(value: string) {
		this.setProperty('order', value);
	}

	public get orientation(): string {
		return this.getPropertyValue('orientation');
	}

	public set orientation(value: string) {
		this.setProperty('orientation', value);
	}

	public get orphans(): string {
		return this.getPropertyValue('orphans');
	}

	public set orphans(value: string) {
		this.setProperty('orphans', value);
	}

	public get outline(): string {
		return this.getPropertyValue('outline');
	}

	public set outline(value: string) {
		this.setProperty('outline', value);
	}

	public get outlineColor(): string {
		return this.getPropertyValue('outline-color');
	}

	public set outlineColor(value: string) {
		this.setProperty('outline-color', value);
	}

	public get outlineOffset(): string {
		return this.getPropertyValue('outline-offset');
	}

	public set outlineOffset(value: string) {
		this.setProperty('outline-offset', value);
	}

	public get outlineStyle(): string {
		return this.getPropertyValue('outline-style');
	}

	public set outlineStyle(value: string) {
		this.setProperty('outline-style', value);
	}

	public get outlineWidth(): string {
		return this.getPropertyValue('outline-width');
	}

	public set outlineWidth(value: string) {
		this.setProperty('outline-width', value);
	}

	public get overflow(): string {
		return this.getPropertyValue('overflow');
	}

	public set overflow(value: string) {
		this.setProperty('overflow', value);
	}

	public get overflowAnchor(): string {
		return this.getPropertyValue('overflow-anchor');
	}

	public set overflowAnchor(value: string) {
		this.setProperty('overflow-anchor', value);
	}

	public get overflowWrap(): string {
		return this.getPropertyValue('overflow-wrap');
	}

	public set overflowWrap(value: string) {
		this.setProperty('overflow-wrap', value);
	}

	public get overflowX(): string {
		return this.getPropertyValue('overflow-x');
	}

	public set overflowX(value: string) {
		this.setProperty('overflow-x', value);
	}

	public get overflowY(): string {
		return this.getPropertyValue('overflow-y');
	}

	public set overflowY(value: string) {
		this.setProperty('overflow-y', value);
	}

	public get overscrollBehavior(): string {
		return this.getPropertyValue('overscroll-behavior');
	}

	public set overscrollBehavior(value: string) {
		this.setProperty('overscroll-behavior', value);
	}

	public get overscrollBehaviorBlock(): string {
		return this.getPropertyValue('overscroll-behavior-block');
	}

	public set overscrollBehaviorBlock(value: string) {
		this.setProperty('overscroll-behavior-block', value);
	}

	public get overscrollBehaviorInline(): string {
		return this.getPropertyValue('overscroll-behavior-inline');
	}

	public set overscrollBehaviorInline(value: string) {
		this.setProperty('overscroll-behavior-inline', value);
	}

	public get overscrollBehaviorX(): string {
		return this.getPropertyValue('overscroll-behavior-x');
	}

	public set overscrollBehaviorX(value: string) {
		this.setProperty('overscroll-behavior-x', value);
	}

	public get overscrollBehaviorY(): string {
		return this.getPropertyValue('overscroll-behavior-y');
	}

	public set overscrollBehaviorY(value: string) {
		this.setProperty('overscroll-behavior-y', value);
	}

	public get overflowClipMargin(): string {
		return this.getPropertyValue('overflow-clip-margin');
	}

	public set overflowClipMargin(value: string) {
		this.setProperty('overflow-clip-margin', value);
	}

	public get padding(): string {
		return this.getPropertyValue('padding');
	}

	public set padding(value: string) {
		this.setProperty('padding', value);
	}

	public get paddingBlockEnd(): string {
		return this.getPropertyValue('padding-block-end');
	}

	public set paddingBlockEnd(value: string) {
		this.setProperty('padding-block-end', value);
	}

	public get paddingBlockStart(): string {
		return this.getPropertyValue('padding-block-start');
	}

	public set paddingBlockStart(value: string) {
		this.setProperty('padding-block-start', value);
	}

	public get paddingBottom(): string {
		return this.getPropertyValue('padding-bottom');
	}

	public set paddingBottom(value: string) {
		this.setProperty('padding-bottom', value);
	}

	public get paddingInlineEnd(): string {
		return this.getPropertyValue('padding-inline-end');
	}

	public set paddingInlineEnd(value: string) {
		this.setProperty('padding-inline-end', value);
	}

	public get paddingInlineStart(): string {
		return this.getPropertyValue('padding-inline-start');
	}

	public set paddingInlineStart(value: string) {
		this.setProperty('padding-inline-start', value);
	}

	public get paddingLeft(): string {
		return this.getPropertyValue('padding-left');
	}

	public set paddingLeft(value: string) {
		this.setProperty('padding-left', value);
	}

	public get paddingRight(): string {
		return this.getPropertyValue('padding-right');
	}

	public set paddingRight(value: string) {
		this.setProperty('padding-right', value);
	}

	public get paddingTop(): string {
		return this.getPropertyValue('padding-top');
	}

	public set paddingTop(value: string) {
		this.setProperty('padding-top', value);
	}

	public get page(): string {
		return this.getPropertyValue('page');
	}

	public set page(value: string) {
		this.setProperty('page', value);
	}

	public get pageBreakAfter(): string {
		return this.getPropertyValue('page-break-after');
	}

	public set pageBreakAfter(value: string) {
		this.setProperty('page-break-after', value);
	}

	public get pageBreakBefore(): string {
		return this.getPropertyValue('page-break-before');
	}

	public set pageBreakBefore(value: string) {
		this.setProperty('page-break-before', value);
	}

	public get pageBreakInside(): string {
		return this.getPropertyValue('page-break-inside');
	}

	public set pageBreakInside(value: string) {
		this.setProperty('page-break-inside', value);
	}

	public get pageOrientation(): string {
		return this.getPropertyValue('page-orientation');
	}

	public set pageOrientation(value: string) {
		this.setProperty('page-orientation', value);
	}

	public get paintOrder(): string {
		return this.getPropertyValue('paint-order');
	}

	public set paintOrder(value: string) {
		this.setProperty('paint-order', value);
	}

	public get perspective(): string {
		return this.getPropertyValue('perspective');
	}

	public set perspective(value: string) {
		this.setProperty('perspective', value);
	}

	public get perspectiveOrigin(): string {
		return this.getPropertyValue('perspective-origin');
	}

	public set perspectiveOrigin(value: string) {
		this.setProperty('perspective-origin', value);
	}

	public get placeContent(): string {
		return this.getPropertyValue('place-content');
	}

	public set placeContent(value: string) {
		this.setProperty('place-content', value);
	}

	public get placeItems(): string {
		return this.getPropertyValue('place-items');
	}

	public set placeItems(value: string) {
		this.setProperty('place-items', value);
	}

	public get placeSelf(): string {
		return this.getPropertyValue('place-self');
	}

	public set placeSelf(value: string) {
		this.setProperty('place-self', value);
	}

	public get pointerEvents(): string {
		return this.getPropertyValue('pointer-events');
	}

	public set pointerEvents(value: string) {
		this.setProperty('pointer-events', value);
	}

	public get position(): string {
		return this.getPropertyValue('position');
	}

	public set position(value: string) {
		this.setProperty('position', value);
	}

	public get quotes(): string {
		return this.getPropertyValue('quotes');
	}

	public set quotes(value: string) {
		this.setProperty('quotes', value);
	}

	public get r(): string {
		return this.getPropertyValue('r');
	}

	public set r(value: string) {
		this.setProperty('r', value);
	}

	public get resize(): string {
		return this.getPropertyValue('resize');
	}

	public set resize(value: string) {
		this.setProperty('resize', value);
	}

	public get right(): string {
		return this.getPropertyValue('right');
	}

	public set right(value: string) {
		this.setProperty('right', value);
	}

	public get rowGap(): string {
		return this.getPropertyValue('row-gap');
	}

	public set rowGap(value: string) {
		this.setProperty('row-gap', value);
	}

	public get rubyPosition(): string {
		return this.getPropertyValue('ruby-position');
	}

	public set rubyPosition(value: string) {
		this.setProperty('ruby-position', value);
	}

	public get rx(): string {
		return this.getPropertyValue('rx');
	}

	public set rx(value: string) {
		this.setProperty('rx', value);
	}

	public get ry(): string {
		return this.getPropertyValue('ry');
	}

	public set ry(value: string) {
		this.setProperty('ry', value);
	}

	public get scrollBehavior(): string {
		return this.getPropertyValue('scroll-behavior');
	}

	public set scrollBehavior(value: string) {
		this.setProperty('scroll-behavior', value);
	}

	public get scrollMargin(): string {
		return this.getPropertyValue('scroll-margin');
	}

	public set scrollMargin(value: string) {
		this.setProperty('scroll-margin', value);
	}

	public get scrollMarginBlock(): string {
		return this.getPropertyValue('scroll-margin-block');
	}

	public set scrollMarginBlock(value: string) {
		this.setProperty('scroll-margin-block', value);
	}

	public get scrollMarginBlockEnd(): string {
		return this.getPropertyValue('scroll-margin-block-end');
	}

	public set scrollMarginBlockEnd(value: string) {
		this.setProperty('scroll-margin-block-end', value);
	}

	public get scrollMarginBlockStart(): string {
		return this.getPropertyValue('scroll-margin-block-start');
	}

	public set scrollMarginBlockStart(value: string) {
		this.setProperty('scroll-margin-block-start', value);
	}

	public get scrollMarginBottom(): string {
		return this.getPropertyValue('scroll-margin-bottom');
	}

	public set scrollMarginBottom(value: string) {
		this.setProperty('scroll-margin-bottom', value);
	}

	public get scrollMarginInline(): string {
		return this.getPropertyValue('scroll-margin-inline');
	}

	public set scrollMarginInline(value: string) {
		this.setProperty('scroll-margin-inline', value);
	}

	public get scrollMarginInlineEnd(): string {
		return this.getPropertyValue('scroll-margin-inline-end');
	}

	public set scrollMarginInlineEnd(value: string) {
		this.setProperty('scroll-margin-inline-end', value);
	}

	public get scrollMarginInlineStart(): string {
		return this.getPropertyValue('scroll-margin-inline-start');
	}

	public set scrollMarginInlineStart(value: string) {
		this.setProperty('scroll-margin-inline-start', value);
	}

	public get scrollMarginLeft(): string {
		return this.getPropertyValue('scroll-margin-left');
	}

	public set scrollMarginLeft(value: string) {
		this.setProperty('scroll-margin-left', value);
	}

	public get scrollMarginRight(): string {
		return this.getPropertyValue('scroll-margin-right');
	}

	public set scrollMarginRight(value: string) {
		this.setProperty('scroll-margin-right', value);
	}

	public get scrollMarginTop(): string {
		return this.getPropertyValue('scroll-margin-top');
	}

	public set scrollMarginTop(value: string) {
		this.setProperty('scroll-margin-top', value);
	}

	public get scrollPadding(): string {
		return this.getPropertyValue('scroll-padding');
	}

	public set scrollPadding(value: string) {
		this.setProperty('scroll-padding', value);
	}

	public get scrollPaddingBlock(): string {
		return this.getPropertyValue('scroll-padding-block');
	}

	public set scrollPaddingBlock(value: string) {
		this.setProperty('scroll-padding-block', value);
	}

	public get scrollPaddingBlockEnd(): string {
		return this.getPropertyValue('scroll-padding-block-end');
	}

	public set scrollPaddingBlockEnd(value: string) {
		this.setProperty('scroll-padding-block-end', value);
	}

	public get scrollPaddingBlockStart(): string {
		return this.getPropertyValue('scroll-padding-block-start');
	}

	public set scrollPaddingBlockStart(value: string) {
		this.setProperty('scroll-padding-block-start', value);
	}

	public get scrollPaddingBottom(): string {
		return this.getPropertyValue('scroll-padding-bottom');
	}

	public set scrollPaddingBottom(value: string) {
		this.setProperty('scroll-padding-bottom', value);
	}

	public get scrollPaddingInline(): string {
		return this.getPropertyValue('scroll-padding-inline');
	}

	public set scrollPaddingInline(value: string) {
		this.setProperty('scroll-padding-inline', value);
	}

	public get scrollPaddingInlineEnd(): string {
		return this.getPropertyValue('scroll-padding-inline-end');
	}

	public set scrollPaddingInlineEnd(value: string) {
		this.setProperty('scroll-padding-inline-end', value);
	}

	public get scrollPaddingInlineStart(): string {
		return this.getPropertyValue('scroll-padding-inline-start');
	}

	public set scrollPaddingInlineStart(value: string) {
		this.setProperty('scroll-padding-inline-start', value);
	}

	public get scrollPaddingLeft(): string {
		return this.getPropertyValue('scroll-padding-left');
	}

	public set scrollPaddingLeft(value: string) {
		this.setProperty('scroll-padding-left', value);
	}

	public get scrollPaddingRight(): string {
		return this.getPropertyValue('scroll-padding-right');
	}

	public set scrollPaddingRight(value: string) {
		this.setProperty('scroll-padding-right', value);
	}

	public get scrollPaddingTop(): string {
		return this.getPropertyValue('scroll-padding-top');
	}

	public set scrollPaddingTop(value: string) {
		this.setProperty('scroll-padding-top', value);
	}

	public get scrollSnapAlign(): string {
		return this.getPropertyValue('scroll-snap-align');
	}

	public set scrollSnapAlign(value: string) {
		this.setProperty('scroll-snap-align', value);
	}

	public get scrollSnapStop(): string {
		return this.getPropertyValue('scroll-snap-stop');
	}

	public set scrollSnapStop(value: string) {
		this.setProperty('scroll-snap-stop', value);
	}

	public get scrollSnapType(): string {
		return this.getPropertyValue('scroll-snap-type');
	}

	public set scrollSnapType(value: string) {
		this.setProperty('scroll-snap-type', value);
	}

	public get shapeImageThreshold(): string {
		return this.getPropertyValue('shape-image-threshold');
	}

	public set shapeImageThreshold(value: string) {
		this.setProperty('shape-image-threshold', value);
	}

	public get shapeMargin(): string {
		return this.getPropertyValue('shape-margin');
	}

	public set shapeMargin(value: string) {
		this.setProperty('shape-margin', value);
	}

	public get shapeOutside(): string {
		return this.getPropertyValue('shape-outside');
	}

	public set shapeOutside(value: string) {
		this.setProperty('shape-outside', value);
	}

	public get shapeRendering(): string {
		return this.getPropertyValue('shape-rendering');
	}

	public set shapeRendering(value: string) {
		this.setProperty('shape-rendering', value);
	}

	public get size(): string {
		return this.getPropertyValue('size');
	}

	public set size(value: string) {
		this.setProperty('size', value);
	}

	public get speak(): string {
		return this.getPropertyValue('speak');
	}

	public set speak(value: string) {
		this.setProperty('speak', value);
	}

	public get src(): string {
		return this.getPropertyValue('src');
	}

	public set src(value: string) {
		this.setProperty('src', value);
	}

	public get stopColor(): string {
		return this.getPropertyValue('stop-color');
	}

	public set stopColor(value: string) {
		this.setProperty('stop-color', value);
	}

	public get stopOpacity(): string {
		return this.getPropertyValue('stop-opacity');
	}

	public set stopOpacity(value: string) {
		this.setProperty('stop-opacity', value);
	}

	public get stroke(): string {
		return this.getPropertyValue('stroke');
	}

	public set stroke(value: string) {
		this.setProperty('stroke', value);
	}

	public get strokeDasharray(): string {
		return this.getPropertyValue('stroke-dasharray');
	}

	public set strokeDasharray(value: string) {
		this.setProperty('stroke-dasharray', value);
	}

	public get strokeDashoffset(): string {
		return this.getPropertyValue('stroke-dashoffset');
	}

	public set strokeDashoffset(value: string) {
		this.setProperty('stroke-dashoffset', value);
	}

	public get strokeLinecap(): string {
		return this.getPropertyValue('stroke-linecap');
	}

	public set strokeLinecap(value: string) {
		this.setProperty('stroke-linecap', value);
	}

	public get strokeLinejoin(): string {
		return this.getPropertyValue('stroke-linejoin');
	}

	public set strokeLinejoin(value: string) {
		this.setProperty('stroke-linejoin', value);
	}

	public get strokeMiterlimit(): string {
		return this.getPropertyValue('stroke-miterlimit');
	}

	public set strokeMiterlimit(value: string) {
		this.setProperty('stroke-miterlimit', value);
	}

	public get strokeOpacity(): string {
		return this.getPropertyValue('stroke-opacity');
	}

	public set strokeOpacity(value: string) {
		this.setProperty('stroke-opacity', value);
	}

	public get strokeWidth(): string {
		return this.getPropertyValue('stroke-width');
	}

	public set strokeWidth(value: string) {
		this.setProperty('stroke-width', value);
	}

	public get syntax(): string {
		return this.getPropertyValue('syntax');
	}

	public set syntax(value: string) {
		this.setProperty('syntax', value);
	}

	public get scrollbarGutter(): string {
		return this.getPropertyValue('scrollbar-gutter');
	}

	public set scrollbarGutter(value: string) {
		this.setProperty('scrollbar-gutter', value);
	}

	public get tabSize(): string {
		return this.getPropertyValue('tab-size');
	}

	public set tabSize(value: string) {
		this.setProperty('tab-size', value);
	}

	public get tableLayout(): string {
		return this.getPropertyValue('table-layout');
	}

	public set tableLayout(value: string) {
		this.setProperty('table-layout', value);
	}

	public get textAlign(): string {
		return this.getPropertyValue('text-align');
	}

	public set textAlign(value: string) {
		this.setProperty('text-align', value);
	}

	public get textAlignLast(): string {
		return this.getPropertyValue('text-align-last');
	}

	public set textAlignLast(value: string) {
		this.setProperty('text-align-last', value);
	}

	public get textAnchor(): string {
		return this.getPropertyValue('text-anchor');
	}

	public set textAnchor(value: string) {
		this.setProperty('text-anchor', value);
	}

	public get textCombineUpright(): string {
		return this.getPropertyValue('text-combine-upright');
	}

	public set textCombineUpright(value: string) {
		this.setProperty('text-combine-upright', value);
	}

	public get textDecoration(): string {
		return this.getPropertyValue('text-decoration');
	}

	public set textDecoration(value: string) {
		this.setProperty('text-decoration', value);
	}

	public get textDecorationColor(): string {
		return this.getPropertyValue('text-decoration-color');
	}

	public set textDecorationColor(value: string) {
		this.setProperty('text-decoration-color', value);
	}

	public get textDecorationLine(): string {
		return this.getPropertyValue('text-decoration-line');
	}

	public set textDecorationLine(value: string) {
		this.setProperty('text-decoration-line', value);
	}

	public get textDecorationSkipInk(): string {
		return this.getPropertyValue('text-decoration-skip-ink');
	}

	public set textDecorationSkipInk(value: string) {
		this.setProperty('text-decoration-skip-ink', value);
	}

	public get textDecorationStyle(): string {
		return this.getPropertyValue('text-decoration-style');
	}

	public set textDecorationStyle(value: string) {
		this.setProperty('text-decoration-style', value);
	}

	public get textIndent(): string {
		return this.getPropertyValue('text-indent');
	}

	public set textIndent(value: string) {
		this.setProperty('text-indent', value);
	}

	public get textOrientation(): string {
		return this.getPropertyValue('text-orientation');
	}

	public set textOrientation(value: string) {
		this.setProperty('text-orientation', value);
	}

	public get textOverflow(): string {
		return this.getPropertyValue('text-overflow');
	}

	public set textOverflow(value: string) {
		this.setProperty('text-overflow', value);
	}

	public get textRendering(): string {
		return this.getPropertyValue('text-rendering');
	}

	public set textRendering(value: string) {
		this.setProperty('text-rendering', value);
	}

	public get textShadow(): string {
		return this.getPropertyValue('text-shadow');
	}

	public set textShadow(value: string) {
		this.setProperty('text-shadow', value);
	}

	public get textSizeAdjust(): string {
		return this.getPropertyValue('text-size-adjust');
	}

	public set textSizeAdjust(value: string) {
		this.setProperty('text-size-adjust', value);
	}

	public get textTransform(): string {
		return this.getPropertyValue('text-transform');
	}

	public set textTransform(value: string) {
		this.setProperty('text-transform', value);
	}

	public get textUnderlinePosition(): string {
		return this.getPropertyValue('text-underline-position');
	}

	public set textUnderlinePosition(value: string) {
		this.setProperty('text-underline-position', value);
	}

	public get top(): string {
		return this.getPropertyValue('top');
	}

	public set top(value: string) {
		this.setProperty('top', value);
	}

	public get touchAction(): string {
		return this.getPropertyValue('touch-action');
	}

	public set touchAction(value: string) {
		this.setProperty('touch-action', value);
	}

	public get transform(): string {
		return this.getPropertyValue('transform');
	}

	public set transform(value: string) {
		this.setProperty('transform', value);
	}

	public get transformBox(): string {
		return this.getPropertyValue('transform-box');
	}

	public set transformBox(value: string) {
		this.setProperty('transform-box', value);
	}

	public get transformOrigin(): string {
		return this.getPropertyValue('transform-origin');
	}

	public set transformOrigin(value: string) {
		this.setProperty('transform-origin', value);
	}

	public get transformStyle(): string {
		return this.getPropertyValue('transform-style');
	}

	public set transformStyle(value: string) {
		this.setProperty('transform-style', value);
	}

	public get transition(): string {
		return this.getPropertyValue('transition');
	}

	public set transition(value: string) {
		this.setProperty('transition', value);
	}

	public get transitionDelay(): string {
		return this.getPropertyValue('transition-delay');
	}

	public set transitionDelay(value: string) {
		this.setProperty('transition-delay', value);
	}

	public get transitionDuration(): string {
		return this.getPropertyValue('transition-duration');
	}

	public set transitionDuration(value: string) {
		this.setProperty('transition-duration', value);
	}

	public get transitionProperty(): string {
		return this.getPropertyValue('transition-property');
	}

	public set transitionProperty(value: string) {
		this.setProperty('transition-property', value);
	}

	public get transitionTimingFunction(): string {
		return this.getPropertyValue('transition-timing-function');
	}

	public set transitionTimingFunction(value: string) {
		this.setProperty('transition-timing-function', value);
	}

	public get textEmphasisColor(): string {
		return this.getPropertyValue('text-emphasis-color');
	}

	public set textEmphasisColor(value: string) {
		this.setProperty('text-emphasis-color', value);
	}

	public get textEmphasisPosition(): string {
		return this.getPropertyValue('text-emphasis-position');
	}

	public set textEmphasisPosition(value: string) {
		this.setProperty('text-emphasis-position', value);
	}

	public get textEmphasisStyle(): string {
		return this.getPropertyValue('text-emphasis-style');
	}

	public set textEmphasisStyle(value: string) {
		this.setProperty('text-emphasis-style', value);
	}

	public get unicodeBidi(): string {
		return this.getPropertyValue('unicode-bidi');
	}

	public set unicodeBidi(value: string) {
		this.setProperty('unicode-bidi', value);
	}

	public get unicodeRange(): string {
		return this.getPropertyValue('unicode-range');
	}

	public set unicodeRange(value: string) {
		this.setProperty('unicode-range', value);
	}

	public get userSelect(): string {
		return this.getPropertyValue('user-select');
	}

	public set userSelect(value: string) {
		this.setProperty('user-select', value);
	}

	public get userZoom(): string {
		return this.getPropertyValue('user-zoom');
	}

	public set userZoom(value: string) {
		this.setProperty('user-zoom', value);
	}

	public get vectorEffect(): string {
		return this.getPropertyValue('vector-effect');
	}

	public set vectorEffect(value: string) {
		this.setProperty('vector-effect', value);
	}

	public get verticalAlign(): string {
		return this.getPropertyValue('vertical-align');
	}

	public set verticalAlign(value: string) {
		this.setProperty('vertical-align', value);
	}

	public get visibility(): string {
		return this.getPropertyValue('visibility');
	}

	public set visibility(value: string) {
		this.setProperty('visibility', value);
	}

	public get whiteSpace(): string {
		return this.getPropertyValue('white-space');
	}

	public set whiteSpace(value: string) {
		this.setProperty('white-space', value);
	}

	public get widows(): string {
		return this.getPropertyValue('widows');
	}

	public set widows(value: string) {
		this.setProperty('widows', value);
	}

	public get width(): string {
		return this.getPropertyValue('width');
	}

	public set width(value: string) {
		this.setProperty('width', value);
	}

	public get willChange(): string {
		return this.getPropertyValue('will-change');
	}

	public set willChange(value: string) {
		this.setProperty('will-change', value);
	}

	public get wordBreak(): string {
		return this.getPropertyValue('word-break');
	}

	public set wordBreak(value: string) {
		this.setProperty('word-break', value);
	}

	public get wordSpacing(): string {
		return this.getPropertyValue('word-spacing');
	}

	public set wordSpacing(value: string) {
		this.setProperty('word-spacing', value);
	}

	public get wordWrap(): string {
		return this.getPropertyValue('word-wrap');
	}

	public set wordWrap(value: string) {
		this.setProperty('word-wrap', value);
	}

	public get writingMode(): string {
		return this.getPropertyValue('writing-mode');
	}

	public set writingMode(value: string) {
		this.setProperty('writing-mode', value);
	}

	public get x(): string {
		return this.getPropertyValue('x');
	}

	public set x(value: string) {
		this.setProperty('x', value);
	}

	public get y(): string {
		return this.getPropertyValue('y');
	}

	public set y(value: string) {
		this.setProperty('y', value);
	}

	public get zIndex(): string {
		return this.getPropertyValue('z-index');
	}

	public set zIndex(value: string) {
		this.setProperty('z-index', value);
	}

	public get zoom(): string {
		return this.getPropertyValue('zoom');
	}

	public set zoom(value: string) {
		this.setProperty('zoom', value);
	}

	public get containerType(): string {
		return this.getPropertyValue('container-type');
	}

	public set containerType(value: string) {
		this.setProperty('container-type', value);
	}

	public get containerName(): string {
		return this.getPropertyValue('container-name');
	}

	public set containerName(value: string) {
		this.setProperty('container-name', value);
	}

	public get aspectRatio(): string {
		return this.getPropertyValue('aspect-ratio');
	}

	public set aspectRatio(value: string) {
		this.setProperty('aspect-ratio', value);
	}

	/* eslint-enable jsdoc/require-jsdoc */

	/**
	 * Returns length.
	 *
	 * @returns Length.
	 */
	public get length(): number {
		return this.#getPropertyManager().size();
	}

	/**
	 * Returns the style decleration as a CSS text.
	 *
	 * @returns CSS text.
	 */
	public get cssText(): string {
		if (this.#element && this.#computed) {
			return '';
		}

		return this.#getPropertyManager().toString();
	}

	/**
	 * Sets CSS text.
	 *
	 * @param cssText CSS text.
	 */
	public set cssText(cssText: string) {
		if (this.#computed) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'cssText' on 'CSSStyleDeclaration': These styles are computed, and the properties are therefore read-only.`,
				DOMExceptionNameEnum.domException
			);
		}

		if (this.#element) {
			this.#cache.propertyManager = new CSSStyleDeclarationPropertyManager({ cssText });
			this.#cache.attributeValue = cssText;
			this.#element.setAttribute('style', this.#cache.propertyManager.toString());
		} else {
			this.#cache.propertyManager = new CSSStyleDeclarationPropertyManager({ cssText });
		}
	}

	/**
	 * Returns item.
	 *
	 * @param index Index.
	 * @returns Item.
	 */
	public item(index: number): string {
		return this.#getPropertyManager().item(index);
	}

	/**
	 * Set a property.
	 *
	 * @param name Property name.
	 * @param value Value. Must not contain "!important" as that should be set using the priority parameter.
	 * @param [priority] Can be "important", or an empty string.
	 */
	public setProperty(name: string, value: string, priority?: 'important' | '' | undefined): void {
		if (this.#computed) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'setProperty' on 'CSSStyleDeclaration': These styles are computed, and therefore the '${name}' property is read-only.`,
				DOMExceptionNameEnum.domException
			);
		}

		if (priority !== '' && priority !== undefined && priority !== 'important') {
			return;
		}

		const stringValue = String(value).trim();
		const propertyManager = this.#getPropertyManager();

		if (stringValue) {
			propertyManager.set(name, stringValue, !!priority);
		} else {
			propertyManager.remove(name);
		}

		if (this.#element) {
			this.#cache.attributeValue = propertyManager.toString();
			if (this.#cache.attributeValue) {
				this.#element.setAttribute('style', this.#cache.attributeValue);
			} else {
				this.#element.removeAttribute('style');
			}
		}
	}

	/**
	 * Removes a property.
	 *
	 * @param name Property name in kebab case.
	 * @param value Value. Must not contain "!important" as that should be set using the priority parameter.
	 * @param [priority] Can be "important", or an empty string.
	 */
	public removeProperty(name: string): void {
		if (this.#computed) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'removeProperty' on 'CSSStyleDeclaration': These styles are computed, and therefore the '${name}' property is read-only.`,
				DOMExceptionNameEnum.domException
			);
		}

		const propertyManager = this.#getPropertyManager();

		propertyManager.remove(name);

		if (this.#element) {
			this.#cache.attributeValue = propertyManager.toString();

			if (this.#cache.attributeValue) {
				this.#element.setAttribute('style', this.#cache.attributeValue);
			} else {
				this.#element.removeAttribute('style');
			}
		}
	}

	/**
	 * Returns a property.
	 *
	 * @param name Property name in kebab case.
	 * @returns Property value.
	 */
	public getPropertyValue(name: string): string {
		return this.#getPropertyManager().get(name)?.value || '';
	}

	/**
	 * Returns a property.
	 *
	 * @param name Property name in kebab case.
	 * @returns "important" if set to be important.
	 */
	public getPropertyPriority(name: string): string {
		return this.#getPropertyManager().get(name)?.important ? 'important' : '';
	}

	/**
	 * Returns property manager.
	 *
	 * @returns Property manager.
	 */
	#getPropertyManager(): CSSStyleDeclarationPropertyManager {
		const element = this.#element;
		const cache = this.#cache;

		if (!element) {
			if (!cache.propertyManager) {
				cache.propertyManager = new CSSStyleDeclarationPropertyManager();
			}
			return cache.propertyManager;
		}

		if (this.#computed) {
			return new CSSStyleDeclarationComputedStyle(element).getComputedStyle();
		}

		const attributeValue = element.getAttribute('style') || '';

		if (cache.attributeValue !== attributeValue) {
			cache.propertyManager = new CSSStyleDeclarationPropertyManager({ cssText: attributeValue });
		}

		return cache.propertyManager;
	}
}
