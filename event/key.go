//AZERTYkeybindings

package event

import "github.com/olup/kobowriter/utils"

var AzertyKeyCode = map[int]string{
	0: "KEY_RESERVED",
	1: "KEY_ESC",

	2:  "&",
	3:  "é",
	4:  "\"",
	5:  "'",
	6:  "(",
	7:  "-",
	8:  "è",
	9:  "_",
	10: "ç",
	11: "à",
	12: ")",
	13: "=",

	14: "KEY_BACKSPACE",
	15: "KEY_TAB",

	16: "a",
	17: "z",
	18: "e",
	19: "r",
	20: "t",
	21: "y",
	22: "u",
	23: "i",
	24: "o",
	25: "p",
	26: "^",
	27: "$",
	28: "KEY_ENTER",
	29: "KEY_L_CTRL",

	30: "q",
	31: "s",
	32: "d",
	33: "f",
	34: "g",
	35: "h",
	36: "j",
	37: "k",
	38: "l",
	39: "m",
	40: "ù",
	41: "*",

	42: "KEY_L_SHIFT",
	43: "<",
	44: "w",
	45: "x",
	46: "c",
	47: "v",
	48: "b",
	49: "n",
	50: ",",
	51: ";",
	52: ":",
	53: "!",
	54: "KEY_R_SHIFT",

	55: "KEY_KPASTERISK",
	56: "KEY_L_ALT",

	57: "KEY_SPACE",
	58: "KEY_CAPSLOCK",
	59: "KEY_F1",
	60: "KEY_F2",
	61: "KEY_F3",
	62: "KEY_F4",
	63: "KEY_F5",
	64: "KEY_F6",
	65: "KEY_F7",
	66: "KEY_F8",
	67: "KEY_F9",
	68: "KEY_F10",

	87: "KEY_F11",
	88: "KEY_F12",

	100: "KEY_ALT_GR",

	103: "KEY_UP",
	105: "KEY_LEFT",
	106: "KEY_RIGHT",
	108: "KEY_DOWN",

	111: "KEY_DEL",

	183: "KEY_F13",
	184: "KEY_F14",
	185: "KEY_F15",
	186: "KEY_F16",
	187: "KEY_F17",
	188: "KEY_F18",
	189: "KEY_F19",
	190: "KEY_F20",
	191: "KEY_F21",
	192: "KEY_F22",
	193: "KEY_F23",
	194: "KEY_F24",
}

var AzertyKeyCodeMaj = map[int]string{
	2:  "1",
	3:  "2",
	4:  "3",
	5:  "4",
	6:  "5",
	7:  "6",
	8:  "7",
	9:  "8",
	10: "9",
	11: "0",
	12: "°",
	13: "+",

	16: "A",
	17: "Z",
	18: "E",
	19: "R",
	20: "T",
	21: "Y",
	22: "U",
	23: "I",
	24: "O",
	25: "P",
	26: "¨",
	27: "£",

	30: "Q",
	31: "S",
	32: "D",
	33: "F",
	34: "G",
	35: "H",
	36: "J",
	37: "K",
	38: "L",
	39: "M",
	40: "%",
	41: "µ",

	43: ">",
	44: "W",
	45: "X",
	46: "C",
	47: "V",
	48: "B",
	49: "N",
	50: "?",
	51: ".",
	52: "/",
	53: "§",
}

var AzertyKeyCodeAltGr = map[int]string{
	3:  "~",
	4:  "#",
	5:  "{",
	6:  "[",
	7:  "|",
	8:  "`",
	9:  "\\",
	10: "^",
	11: "@",
	12: "]",
	13: "}",

	16: "æ",
	17: "«",
	18: "€",
	19: "¶",
	20: "ŧ",
	21: "←",
	22: "↓",
	23: "→",
	24: "ø",
	25: "þ",
	26: "¨",
	27: "¤",

	30: "@",
	31: "ß",
	32: "ð",
	33: "đ",
	34: "ŋ",
	35: "ħ",

	37: "ĸ",
	38: "ł",
	39: "µ",

	41: "`",

	43: "|",
	44: "ł",
	45: "»",
	46: "¢",
	47: "“",
	48: "”",
	49: "n",
	50: "´",
	51: "─",
	52: "·",
}

func GetKeyMapMaj(lang string) map[int]string {
	if lang == utils.QWERTY {
		return QwertyKeyCodeMaj
	} else {
		return AzertyKeyCodeMaj
	}
}
func GetKeyMapAltGr(lang string) map[int]string {
	if lang == utils.QWERTY {
		return QwertyKeyCodeAltGr
	} else {
		return AzertyKeyCodeAltGr
	}
}
func GetKeyMap(lang string) map[int]string {
	if lang == utils.QWERTY {
		return QwertyKeyCode
	} else {
		return AzertyKeyCode
	}
}
