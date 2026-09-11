package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/types"
	"github.com/nyaruka/phonenumbers"
)

// waMetaTimezoneLocations maps Meta timezone_id values to IANA locations.
// Source: https://gist.github.com/italomlp/0d83002e4cee98e22d2db6051649a667
// Keep the IANA location rather than a fixed offset: its offset can change with DST.
var waMetaTimezoneLocations = map[int]string{
	1: "America/Los_Angeles", 2: "America/Denver", 3: "Pacific/Honolulu", 4: "America/Anchorage", 5: "America/Phoenix", 6: "America/Chicago", 7: "America/New_York", 8: "Asia/Dubai",
	9: "America/Argentina/San_Luis", 10: "America/Argentina/Buenos_Aires", 11: "America/Argentina/Salta", 12: "Europe/Vienna", 13: "Australia/Perth", 14: "Australia/Broken_Hill", 15: "Australia/Sydney", 16: "Europe/Sarajevo",
	17: "Asia/Dhaka", 18: "Europe/Brussels", 19: "Europe/Sofia", 20: "Asia/Bahrain", 21: "America/La_Paz", 22: "America/Noronha", 23: "America/Campo_Grande", 24: "America/Belem", 25: "America/Sao_Paulo", 26: "America/Nassau", 27: "America/Dawson", 28: "America/Vancouver", 29: "America/Dawson_Creek", 30: "America/Edmonton", 31: "America/Rainy_River", 32: "America/Regina", 33: "America/Atikokan", 34: "America/Iqaluit", 35: "America/Toronto", 36: "America/Blanc-Sablon", 37: "America/Halifax", 38: "America/St_Johns", 39: "Europe/Zurich", 40: "Pacific/Easter", 41: "America/Santiago", 42: "Asia/Shanghai", 43: "America/Bogota", 44: "America/Costa_Rica", 45: "Asia/Nicosia", 46: "Europe/Prague", 47: "Europe/Berlin", 48: "Europe/Copenhagen", 49: "America/Santo_Domingo", 50: "Pacific/Galapagos", 51: "America/Guayaquil", 52: "Europe/Tallinn", 53: "Africa/Cairo", 54: "Atlantic/Canary", 55: "Europe/Madrid", 56: "Europe/Helsinki", 57: "Europe/Paris", 58: "Europe/London", 59: "Africa/Accra", 60: "Europe/Athens", 61: "America/Guatemala", 62: "Asia/Hong_Kong", 63: "America/Tegucigalpa", 64: "Europe/Zagreb", 65: "Europe/Budapest", 66: "Asia/Jakarta", 67: "Asia/Makassar", 68: "Asia/Jayapura", 69: "Europe/Dublin", 70: "Asia/Jerusalem", 71: "Asia/Kolkata", 72: "Asia/Baghdad", 73: "Atlantic/Reykjavik", 74: "Europe/Rome", 75: "America/Jamaica", 76: "Asia/Amman", 77: "Asia/Tokyo", 78: "Africa/Nairobi", 79: "Asia/Seoul", 80: "Asia/Kuwait", 81: "Asia/Beirut", 82: "Asia/Colombo", 83: "Europe/Vilnius", 84: "Europe/Luxembourg", 85: "Europe/Riga", 86: "Africa/Casablanca", 87: "Europe/Skopje", 88: "Europe/Malta", 89: "Indian/Mauritius", 90: "Indian/Maldives", 91: "America/Tijuana", 92: "America/Hermosillo", 93: "America/Mazatlan", 94: "America/Mexico_City", 95: "Asia/Kuala_Lumpur", 96: "Africa/Lagos", 97: "America/Managua", 98: "Europe/Amsterdam", 99: "Europe/Oslo", 100: "Pacific/Auckland", 101: "Asia/Muscat", 102: "America/Panama", 103: "America/Lima", 104: "Asia/Manila", 105: "Asia/Karachi", 106: "Europe/Warsaw", 107: "America/Puerto_Rico", 108: "Asia/Gaza", 109: "Atlantic/Azores", 110: "Europe/Lisbon", 111: "America/Asuncion", 112: "Asia/Qatar", 113: "Europe/Bucharest", 114: "Europe/Belgrade", 115: "Europe/Kaliningrad", 116: "Europe/Moscow", 117: "Europe/Samara", 118: "Asia/Yekaterinburg", 119: "Asia/Omsk", 120: "Asia/Krasnoyarsk", 121: "Asia/Irkutsk", 122: "Asia/Yakutsk", 123: "Asia/Vladivostok", 124: "Asia/Magadan", 125: "Asia/Kamchatka", 126: "Asia/Riyadh", 127: "Europe/Stockholm", 128: "Asia/Singapore", 129: "Europe/Ljubljana", 130: "Europe/Bratislava", 131: "America/El_Salvador", 132: "Asia/Bangkok", 133: "Africa/Tunis", 134: "Europe/Istanbul", 135: "America/Port_of_Spain", 136: "Asia/Taipei", 137: "Europe/Kyiv", 138: "America/Montevideo", 139: "America/Caracas", 140: "Asia/Ho_Chi_Minh", 141: "Africa/Johannesburg", 142: "America/Winnipeg", 143: "America/Detroit", 144: "Australia/Melbourne", 145: "Asia/Kathmandu", 146: "Asia/Baku", 147: "Africa/Abidjan", 148: "Africa/Addis_Ababa", 149: "Africa/Algiers", 150: "Africa/Asmara", 151: "Africa/Bamako", 152: "Africa/Bangui", 153: "Africa/Banjul", 154: "Africa/Bissau", 155: "Africa/Blantyre", 156: "Africa/Brazzaville", 157: "Africa/Bujumbura", 158: "Africa/Ceuta", 159: "Africa/Conakry", 160: "Africa/Dakar", 161: "Africa/Dar_Es_Salaam", 162: "Africa/Djibouti", 163: "Africa/Douala", 164: "Africa/El_Aaiun", 165: "Africa/Freetown", 166: "Africa/Gaborone", 167: "Africa/Harare", 168: "Africa/Juba", 169: "Africa/Kampala", 170: "Africa/Khartoum", 171: "Africa/Kigali", 172: "Africa/Kinshasa", 173: "Africa/Libreville", 174: "Africa/Lome", 175: "Africa/Luanda", 176: "Africa/Lubumbashi", 177: "Africa/Lusaka", 178: "Africa/Malabo", 179: "Africa/Maputo", 180: "Africa/Maseru", 181: "Africa/Mbabane", 182: "Africa/Mogadishu", 183: "Africa/Monrovia", 184: "Africa/Ndjamena", 185: "Africa/Niamey", 186: "Africa/Nouakchott", 187: "Africa/Ouagadougou", 188: "Africa/Porto-Novo", 189: "Africa/Sao_Tome", 190: "Africa/Tripoli", 191: "Africa/Windhoek",
	192: "America/Adak", 193: "America/Anguilla", 194: "America/Antigua", 195: "America/Araguaina", 196: "America/Argentina/Catamarca", 197: "America/Argentina/Cordoba", 198: "America/Argentina/Jujuy", 199: "America/Argentina/La_Rioja", 200: "America/Argentina/Mendoza", 201: "America/Argentina/Rio_Gallegos", 202: "America/Argentina/San_Juan", 203: "America/Argentina/Tucuman", 204: "America/Argentina/Ushuaia", 205: "America/Aruba", 206: "America/Bahia", 207: "America/Bahia_Banderas", 208: "America/Barbados", 209: "America/Belize", 210: "America/Boa_Vista", 211: "America/Boise", 212: "America/Cambridge_Bay", 213: "America/Cancun", 214: "America/Cayenne", 215: "America/Cayman", 216: "America/Chihuahua", 217: "America/Creston", 218: "America/Cuiaba", 219: "America/Curacao", 220: "America/Danmarkshavn", 221: "America/Dominica", 222: "America/Eirunepe", 223: "America/Fort_Nelson", 224: "America/Fortaleza", 225: "America/Glace_Bay", 226: "America/Nuuk", 227: "America/Goose_Bay", 228: "America/Grand_Turk", 229: "America/Grenada", 230: "America/Guadeloupe", 231: "America/Guyana", 232: "America/Havana", 233: "America/Indiana/Indianapolis", 234: "America/Indiana/Knox", 235: "America/Indiana/Marengo", 236: "America/Indiana/Petersburg", 237: "America/Indiana/Tell_City", 238: "America/Indiana/Vevay", 239: "America/Indiana/Vincennes", 240: "America/Indiana/Winamac", 241: "America/Indianapolis", 242: "America/Inuvik", 243: "America/Juneau", 244: "America/Kentucky/Louisville", 245: "America/Kentucky/Monticello", 246: "America/Kralendijk", 247: "America/Lower_Princes", 248: "America/Maceio", 249: "America/Manaus", 250: "America/Marigot", 251: "America/Martinique", 252: "America/Matamoros", 253: "America/Menominee", 254: "America/Merida", 255: "America/Metlakatla", 256: "America/Miquelon", 257: "America/Moncton", 258: "America/Monterrey", 259: "America/Montreal", 260: "America/Montserrat", 261: "America/Nipigon", 262: "America/Nome", 263: "America/North_Dakota/Beulah", 264: "America/North_Dakota/Center", 265: "America/North_Dakota/New_Salem", 266: "America/Ojinaga", 267: "America/Pangnirtung", 268: "America/Paramaribo", 269: "America/Port-au-Prince", 270: "America/Porto_Velho", 271: "America/Punta_Arenas", 272: "America/Rankin_Inlet", 273: "America/Recife", 274: "America/Resolute", 275: "America/Rio_Branco", 276: "America/Santarem", 277: "America/Scoresbysund", 278: "America/Sitka", 279: "America/St_Barthelemy", 280: "America/St_Kitts", 281: "America/St_Lucia", 282: "America/St_Thomas", 283: "America/St_Vincent", 284: "America/Swift_Current", 285: "America/Thule", 286: "America/Thunder_Bay", 287: "America/Tortola", 288: "America/Whitehorse", 289: "America/Yakutat", 290: "America/Yellowknife",
	291: "Antarctica/Casey", 292: "Antarctica/Davis", 293: "Antarctica/DumontDUrville", 294: "Antarctica/Macquarie", 295: "Antarctica/Mawson", 296: "Antarctica/McMurdo", 297: "Antarctica/Palmer", 298: "Antarctica/Rothera", 299: "Antarctica/Syowa", 300: "Antarctica/Troll", 301: "Antarctica/Vostok", 302: "Arctic/Longyearbyen", 303: "Asia/Aden", 304: "Asia/Almaty", 305: "Asia/Anadyr", 306: "Asia/Aqtau", 307: "Asia/Aqtobe", 308: "Asia/Ashgabat", 309: "Asia/Atyrau", 310: "Asia/Barnaul", 311: "Asia/Bishkek", 312: "Asia/Brunei", 313: "Asia/Chita", 314: "Asia/Choibalsan", 315: "Asia/Damascus", 316: "Asia/Dili", 317: "Asia/Dushanbe", 318: "Asia/Famagusta", 319: "Asia/Hebron", 320: "Asia/Hovd", 321: "Asia/Istanbul", 322: "Asia/Kabul", 323: "Asia/Khandyga", 324: "Asia/Kuching", 325: "Asia/Macau", 326: "Asia/Novokuznetsk", 327: "Asia/Novosibirsk", 328: "Asia/Oral", 329: "Asia/Phnom_Penh", 330: "Asia/Pontianak", 331: "Asia/Pyongyang", 332: "Asia/Qostanay", 333: "Asia/Qyzylorda", 334: "Asia/Sakhalin", 335: "Asia/Samarkand", 336: "Asia/Srednekolymsk", 337: "Asia/Tashkent", 338: "Asia/Tbilisi", 339: "Asia/Tehran", 340: "Asia/Thimphu", 341: "Asia/Tomsk", 342: "Asia/Ulaanbaatar", 343: "Asia/Urumqi", 344: "Asia/Ust-Nera", 345: "Asia/Vientiane", 346: "Asia/Yangon", 347: "Asia/Yerevan", 348: "Atlantic/Bermuda", 349: "Atlantic/Cape_Verde", 350: "Atlantic/Faroe", 351: "Atlantic/Madeira", 352: "Atlantic/South_Georgia", 353: "Atlantic/St_Helena", 354: "Atlantic/Stanley", 355: "Australia/Adelaide", 356: "Australia/Brisbane", 357: "Australia/Currie", 358: "Australia/Darwin", 359: "Australia/Eucla", 360: "Australia/Hobart", 361: "Australia/Lindeman", 362: "Australia/Lord_Howe", 363: "CET", 364: "CST6CDT", 365: "EET", 366: "EST", 367: "EST5EDT", 368: "Etc/GMT", 369: "Etc/GMT+0", 370: "Etc/GMT+1", 371: "Etc/GMT+10", 372: "Etc/GMT+11", 373: "Etc/GMT+12", 374: "Etc/GMT+2", 375: "Etc/GMT+3", 376: "Etc/GMT+4", 377: "Etc/GMT+5", 378: "Etc/GMT+6", 379: "Etc/GMT+7", 380: "Etc/GMT+8", 381: "Etc/GMT+9", 382: "Etc/GMT-0", 383: "Etc/GMT-1", 384: "Etc/GMT-10", 385: "Etc/GMT-11", 386: "Etc/GMT-12", 387: "Etc/GMT-13", 388: "Etc/GMT-14", 389: "Etc/GMT-2", 390: "Etc/GMT-3", 391: "Etc/GMT-4", 392: "Etc/GMT-5", 393: "Etc/GMT-6", 394: "Etc/GMT-7", 395: "Etc/GMT-8", 396: "Etc/GMT-9", 397: "Etc/GMT0", 398: "Etc/Greenwich", 399: "Etc/Universal", 400: "Etc/Zulu",
	401: "Europe/Andorra", 402: "Europe/Astrakhan", 403: "Europe/Busingen", 404: "Europe/Chisinau", 405: "Europe/Gibraltar", 406: "Europe/Guernsey", 407: "Europe/Isle_of_Man", 408: "Europe/Jersey", 409: "Europe/Kirov", 410: "Europe/Mariehamn", 411: "Europe/Minsk", 412: "Europe/Monaco", 413: "Europe/Nicosia", 414: "Europe/Podgorica", 415: "Europe/San_Marino", 416: "Europe/Saratov", 417: "Europe/Simferopol", 418: "Europe/Tirane", 419: "Europe/Ulyanovsk", 420: "Europe/Uzhgorod", 421: "Europe/Vaduz", 422: "Europe/Vatican", 423: "Europe/Volgograd", 424: "Europe/Zaporozhye", 425: "GMT", 426: "HST", 427: "Indian/Antananarivo", 428: "Indian/Chagos", 429: "Indian/Christmas", 430: "Indian/Cocos", 431: "Indian/Comoro", 432: "Indian/Kerguelen", 433: "Indian/Mahe", 434: "Indian/Mayotte", 435: "Indian/Reunion", 436: "MET", 437: "MST", 438: "MST7MDT", 439: "PST8PDT", 440: "Pacific/Apia", 441: "Pacific/Bougainville", 442: "Pacific/Chatham", 443: "Pacific/Chuuk", 444: "Pacific/Efate", 445: "Pacific/Enderbury", 446: "Pacific/Fakaofo", 447: "Pacific/Fiji", 448: "Pacific/Funafuti", 449: "Pacific/Gambier", 450: "Pacific/Guadalcanal", 451: "Pacific/Guam", 452: "Pacific/Kiritimati", 453: "Pacific/Kosrae", 454: "Pacific/Kwajalein", 455: "Pacific/Majuro", 456: "Pacific/Marquesas", 457: "Pacific/Midway", 458: "Pacific/Nauru", 459: "Pacific/Niue", 460: "Pacific/Norfolk", 461: "Pacific/Noumea", 462: "Pacific/Pago_Pago", 463: "Pacific/Palau", 464: "Pacific/Pitcairn", 465: "Pacific/Pohnpei", 466: "Pacific/Port_Moresby", 467: "Pacific/Rarotonga", 468: "Pacific/Saipan", 469: "Pacific/Tahiti", 470: "Pacific/Tarawa", 471: "Pacific/Tongatapu", 472: "Pacific/Wake", 473: "Pacific/Wallis", 474: "UTC", 475: "WET", 476: "Asia/Kolkata", 477: "Asia/Kathmandu", 478: "America/Nuuk",
}

// GetWAGMTOffset returns the current GMT offset for a Meta WhatsApp timezone_id.
func GetWAGMTOffset(timezoneID string) (string, error) {
	return getWAGMTOffsetAt(timezoneID, time.Now())
}

func getWAGMTOffsetAt(timezoneID string, at time.Time) (string, error) {
	id, err := strconv.Atoi(strings.TrimSpace(timezoneID))
	if err != nil {
		return "", fmt.Errorf("invalid Meta timezone ID %q: %w", timezoneID, err)
	}
	locationName, ok := waMetaTimezoneLocations[id]
	if !ok {
		return "", fmt.Errorf("unknown Meta timezone ID %d", id)
	}
	location, err := time.LoadLocation(locationName)
	if err != nil {
		return "", fmt.Errorf("load timezone %q for Meta timezone ID %d: %w", locationName, id, err)
	}
	_, offsetSeconds := at.In(location).Zone()
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	return fmt.Sprintf("GMT%s%02d:%02d", sign, offsetSeconds/3600, offsetSeconds%3600/60), nil
}

func NormalizeWAId(phoneNumber string) string {
	waId := strings.ReplaceAll(phoneNumber, "+", "")
	waId = strings.ReplaceAll(waId, " ", "")
	waId = strings.ReplaceAll(waId, "-", "")
	return waId
}

func GetChatChannelName(phoneNumberId string, customerWAId string, customerMetaUserId string) string {
	customerId := customerWAId
	if customerId == "" {
		customerId = customerMetaUserId
	}
	return fmt.Sprintf("chat:%s:%s", phoneNumberId, customerId)
}

// take X-Hub-Signature-256 from header
func VerifyWhatsAppWebhookSignature(signature string, body []byte, appSecret string) bool {
	signature = strings.TrimPrefix(strings.TrimSpace(signature), "sha256=")
	if signature == "" || strings.TrimSpace(appSecret) == "" {
		return false
	}
	providedSignature, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	_, _ = mac.Write(body)
	return hmac.Equal(providedSignature, mac.Sum(nil))
}

func CanTransitionWAMessageStatus(current types.WAMessageStatus, next types.WAMessageStatus) bool {
	if current == next {
		return false
	}
	if next == types.WAMessageStatusFailed {
		switch current {
		case types.WAMessageStatusAccepted, types.WAMessageStatusHeldForQualityAssessment, types.WAMessageStatusPaused, types.WAMessageStatusSent:
			return true
		default:
			return false
		}
	}
	if current == types.WAMessageStatusFailed {
		return false
	}
	return waMessageStatusRank(next) > waMessageStatusRank(current)
}

func waMessageStatusRank(status types.WAMessageStatus) int {
	switch status {
	case types.WAMessageStatusAccepted, types.WAMessageStatusHeldForQualityAssessment, types.WAMessageStatusPaused:
		return 1
	case types.WAMessageStatusSent:
		return 2
	case types.WAMessageStatusDelivered:
		return 3
	case types.WAMessageStatusRead:
		return 4
	case types.WAMessageStatusPlayed:
		return 5
	default:
		return 0
	}
}

// waid = 6590000000
func GetCountryCodeAndPhoneNumberFromWAId(waID string) (string, string, error) {
	phoneNumber, err := phonenumbers.Parse("+"+waID, "")
	if err != nil || !phonenumbers.IsValidNumber(phoneNumber) {
		return "", "", fmt.Errorf("invalid WhatsApp ID %q", waID)
	}
	return fmt.Sprintf("%d", phoneNumber.GetCountryCode()), phonenumbers.GetNationalSignificantNumber(phoneNumber), nil
}
