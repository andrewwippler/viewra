package parsers

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mantonx/viewra/internal/domain/scanner"
)

// Movie filename patterns to extract title and year
var moviePatterns = []*regexp.Regexp{
	// Title with year in parentheses: "The Matrix (1999).mkv"
	regexp.MustCompile(`(?i)^(.+?)\s*[\(\[](\d{4})[\)\]]`),

	// Title with year separated by dots/spaces: "Inception.2010.1080p.mkv"
	regexp.MustCompile(`(?i)^(.+?)[\s._-]+(\d{4})[\s._-]`),

	// Title with year at the end: "The Godfather 1972.mkv"
	regexp.MustCompile(`(?i)^(.+?)\s+(\d{4})$`),
}

// Movie quality/resolution patterns
var resolutionPatterns = map[string]string{
	"2160p": "4K",
	"1080p": "1080p",
	"720p":  "720p",
	"480p":  "480p",
	"4K":    "4K",
	"UHD":   "4K",
}

// Movie quality source patterns
var qualityPatterns = []string{
	"Remux", "BluRay", "BRRip", "BDRip", "WEBRip", "WEB-DL", "HDTV", "DVDRip",
	"WEBDL", "Bluray", "BRRIP", "WEBRIP", "Bluray",
}

// ParseMovie extracts movie metadata from a filename.
// It attempts to extract the title, year, resolution, quality, IMDb ID, and language from the filename.
func ParseMovie(filename string) (*scanner.MovieInfo, error) {
	// Get the filename without extension
	filenameNoExt := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	info := &scanner.MovieInfo{}

	// Extract IMDb ID if present (e.g., [imdbid-tt1234567])
	info.ImdbID = extractImdbID(filenameNoExt)

	// Extract edition info (e.g., "Extended Cut", "Director's Cut", "Remastered")
	info.Edition = extractEdition(filenameNoExt)

	// Extract language from filename (e.g., Movie.2020.eng.mkv, Movie.2020.[fre].mkv)
	// Must be done before title cleaning (language tags are removed from the title)
	info.Language = extractLanguage(filenameNoExt)

	// Try each pattern to extract title and year
	var titleRaw string
	yearFound := false

	for _, pattern := range moviePatterns {
		matches := pattern.FindStringSubmatch(filenameNoExt)
		if len(matches) >= 3 {
			titleRaw = matches[1]
			year, err := strconv.Atoi(matches[2])
			if err == nil && isValidMovieYear(year) {
				info.Year = year
				yearFound = true
				break
			}
		}
	}

	// If no year found with patterns, extract title without year
	if !yearFound {
		titleRaw = extractTitleWithoutYear(filenameNoExt)
	}

	// Clean the title
	info.Title = cleanMovieTitle(titleRaw)
	if info.Title == "" {
		return nil, fmt.Errorf("unable to extract movie title from: %s", filename)
	}

	// Extract resolution and quality
	info.Resolution = extractResolution(filenameNoExt)
	info.Quality = extractQuality(filenameNoExt)

	return info, nil
}

// extractTitleWithoutYear extracts the title when no year pattern is found
// It removes everything after quality/resolution tags
func extractTitleWithoutYear(filename string) string {
	filenameLower := strings.ToLower(filename)

	// Find the earliest occurrence of quality tags
	earliestIdx := len(filename)

	// Check for resolution tags
	for tag := range resolutionPatterns {
		tagLower := strings.ToLower(tag)
		if idx := strings.Index(filenameLower, tagLower); idx != -1 && idx < earliestIdx {
			earliestIdx = idx
		}
	}

	// Check for quality tags
	for _, tag := range qualityPatterns {
		tagLower := strings.ToLower(tag)
		if idx := strings.Index(filenameLower, tagLower); idx != -1 && idx < earliestIdx {
			earliestIdx = idx
		}
	}

	// Check for codec tags
	for _, tag := range qualityTags {
		tagLower := strings.ToLower(tag)
		if idx := strings.Index(filenameLower, tagLower); idx != -1 && idx < earliestIdx {
			earliestIdx = idx
		}
	}

	// If we found a tag, extract everything before it
	if earliestIdx < len(filename) {
		return filename[:earliestIdx]
	}

	return filename
}

// cleanMovieTitle removes quality tags, resolution, language tags, and normalizes the title
func cleanMovieTitle(title string) string {
	// Replace dots and underscores with spaces
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Remove common brackets and their content (includes language tags like [eng] {fre})
	title = regexp.MustCompile(`\[.*?\]`).ReplaceAllString(title, "")
	title = regexp.MustCompile(`\{.*?\}`).ReplaceAllString(title, "")

	// Remove resolution tags
	titleLower := strings.ToLower(title)
	for tag := range resolutionPatterns {
		tagLower := strings.ToLower(tag)
		if idx := strings.Index(titleLower, tagLower); idx != -1 {
			title = title[:idx]
			titleLower = titleLower[:idx]
		}
	}

	// Remove quality tags
	for _, tag := range qualityPatterns {
		tagLower := strings.ToLower(tag)
		if idx := strings.Index(titleLower, tagLower); idx != -1 {
			title = title[:idx]
			titleLower = titleLower[:idx]
		}
	}

	// Remove codec and other quality tags
	for _, tag := range qualityTags {
		tagLower := strings.ToLower(tag)
		if idx := strings.Index(titleLower, tagLower); idx != -1 {
			title = title[:idx]
			titleLower = titleLower[:idx]
		}
	}

	// Remove known ISO 639-2 language codes that appear as space-separated tokens
	// (e.g., "Movie 2020 eng" → "Movie 2020")
	titleLower = strings.ToLower(title)
	langRemoved := false
	titleWords := strings.Fields(title)
	var cleaned []string
	for _, word := range titleWords {
		wordLower := strings.ToLower(word)
		if len(wordLower) == 3 {
			if _, ok := iso6392Codes[wordLower]; ok {
				langRemoved = true
				continue
			}
		}
		cleaned = append(cleaned, word)
	}
	if langRemoved {
		title = strings.Join(cleaned, " ")
	}

	// Clean up whitespace
	title = strings.TrimSpace(title)
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")

	// Remove trailing dashes or dots
	title = strings.TrimRight(title, " -.")

	return title
}

// extractResolution extracts the video resolution from the filename
func extractResolution(filename string) string {
	filenameLower := strings.ToLower(filename)

	for tag, resolution := range resolutionPatterns {
		tagLower := strings.ToLower(tag)
		if strings.Contains(filenameLower, tagLower) {
			return resolution
		}
	}

	return ""
}

// extractQuality extracts the source quality from the filename
func extractQuality(filename string) string {
	filenameLower := strings.ToLower(filename)

	for _, quality := range qualityPatterns {
		qualityLower := strings.ToLower(quality)
		if strings.Contains(filenameLower, qualityLower) {
			return quality
		}
	}

	return ""
}

// isValidMovieYear checks if a year is within a reasonable range for movies
func isValidMovieYear(year int) bool {
	return year >= 1880 && year <= 2100
}

// extractImdbID extracts the IMDb ID from the filename
// Looks for patterns like [imdbid-tt1234567] or [tt1234567]
func extractImdbID(filename string) string {
	// Pattern: [imdbid-tt1234567] or [tt1234567]
	imdbPattern := regexp.MustCompile(`\[(?:imdbid-)?([Tt][Tt]\d{7,8})\]`)
	matches := imdbPattern.FindStringSubmatch(filename)
	if len(matches) >= 2 {
		return strings.ToLower(matches[1])
	}
	return ""
}

// iso6392Codes is a set of ISO 639-2/B three-letter language codes commonly used in media filenames.
// These are matched against filename tokens to determine the file's language.
var iso6392Codes = map[string]string{
	"aar": "aar", "abk": "abk", "ace": "ace", "ach": "ach", "ada": "ada",
	"ady": "ady", "afa": "afa", "afh": "afh", "afr": "afr", "aka": "aka",
	"akk": "akk", "ale": "ale", "alg": "alg", "amh": "amh", "ang": "ang",
	"apa": "apa", "ara": "ara", "arc": "arc", "arg": "arg", "arm": "arm",
	"arn": "arn", "arp": "arp", "art": "art", "arw": "arw", "asm": "asm",
	"ast": "ast", "ath": "ath", "aus": "aus", "ava": "ava", "ave": "ave",
	"awa": "awa", "aym": "aym", "aze": "aze", "bad": "bad", "bai": "bai",
	"bak": "bak", "bal": "bal", "bam": "bam", "ban": "ban", "baq": "baq",
	"bas": "bas", "bat": "bat", "bej": "bej", "bel": "bel", "bem": "bem",
	"ben": "ben", "ber": "ber", "bho": "bho", "bih": "bih", "bik": "bik",
	"bin": "bin", "bis": "bis", "bla": "bla", "bnt": "bnt", "bos": "bos",
	"bra": "bra", "bre": "bre", "btk": "btk", "bua": "bua", "bug": "bug",
	"bul": "bul", "bur": "bur", "cad": "cad", "cai": "cai", "car": "car",
	"cat": "cat", "cau": "cau", "ceb": "ceb", "cel": "cel", "cha": "cha",
	"chb": "chb", "che": "che", "chg": "chg", "chi": "chi", "chk": "chk",
	"chm": "chm", "chn": "chn", "cho": "cho", "chp": "chp", "chr": "chr",
	"chu": "chu", "chv": "chv", "chy": "chy", "cmc": "cmc", "cop": "cop",
	"cor": "cor", "cos": "cos", "cpe": "cpe", "cpf": "cpf", "cpp": "cpp",
	"cre": "cre", "crh": "crh", "crp": "crp", "cus": "cus", "cze": "cze",
	"dak": "dak", "dan": "dan", "dar": "dar", "day": "day", "del": "del",
	"den": "den", "deu": "deu", "dgr": "dgr", "din": "din", "div": "div",
	"doi": "doi", "dra": "dra", "dua": "dua", "dum": "dum", "dut": "dut",
	"dyu": "dyu", "dzo": "dzo", "efi": "efi", "egy": "egy", "eka": "eka",
	"elo": "elo", "eng": "eng", "enm": "enm", "epo": "epo", "est": "est",
	"eth": "eth", "eve": "eve", "ewe": "ewe", "ewo": "ewo", "fan": "fan",
	"fao": "fao", "far": "far", "fat": "fat", "fij": "fij", "fil": "fil",
	"fin": "fin", "fiu": "fiu", "fon": "fon", "fre": "fre", "frm": "frm",
	"fro": "fro", "fry": "fry", "ful": "ful", "fur": "fur", "gaa": "gaa",
	"gay": "gay", "gba": "gba", "gem": "gem", "geo": "geo", "ger": "ger",
	"gez": "gez", "gil": "gil", "gla": "gla", "gle": "gle", "glg": "glg",
	"glv": "glv", "gme": "gme", "gmh": "gmh", "goh": "goh", "gon": "gon",
	"gor": "gor", "got": "got", "grb": "grb", "grc": "grc", "gre": "gre",
	"grn": "grn", "gsw": "gsw", "guj": "guj", "gwi": "gwi", "hai": "hai",
	"hap": "hap", "hau": "hau", "haw": "haw", "heb": "heb", "her": "her",
	"hil": "hil", "him": "him", "hin": "hin", "hit": "hit", "hmn": "hmn",
	"hmo": "hmo", "hrv": "hrv", "hsb": "hsb", "hun": "hun", "hup": "hup",
	"iba": "iba", "ibo": "ibo", "ice": "ice", "ido": "ido", "iii": "iii",
	"ijo": "ijo", "iku": "iku", "ile": "ile", "ilo": "ilo", "ina": "ina",
	"inc": "inc", "ind": "ind", "ine": "ine", "inh": "inh", "ipk": "ipk",
	"ira": "ira", "iro": "iro", "ita": "ita", "jav": "jav", "jpn": "jpn",
	"jpr": "jpr", "jrb": "jrb", "kaa": "kaa", "kab": "kab", "kac": "kac",
	"kal": "kal", "kam": "kam", "kan": "kan", "kar": "kar", "kas": "kas",
	"kau": "kau", "kaw": "kaw", "kaz": "kaz", "kbd": "kbd", "kha": "kha",
	"khi": "khi", "khm": "khm", "kho": "kho", "kik": "kik", "kin": "kin",
	"kir": "kir", "kmb": "kmb", "kok": "kok", "kom": "kom", "kon": "kon",
	"kor": "kor", "kos": "kos", "kpe": "kpe", "krc": "krc", "kro": "kro",
	"kru": "kru", "kua": "kua", "kum": "kum", "kur": "kur", "kut": "kut",
	"lad": "lad", "lah": "lah", "lam": "lam", "lao": "lao", "lat": "lat",
	"lav": "lav", "lez": "lez", "lim": "lim", "lin": "lin", "lit": "lit",
	"lol": "lol", "loz": "loz", "ltz": "ltz", "lua": "lua", "lub": "lub",
	"lug": "lug", "lui": "lui", "lun": "lun", "luo": "luo", "lus": "lus",
	"mac": "mac", "mad": "mad", "mag": "mag", "mah": "mah", "mai": "mai",
	"mak": "mak", "mal": "mal", "man": "man", "mao": "mao", "map": "map",
	"mar": "mar", "mas": "mas", "may": "may", "mdf": "mdf", "mdr": "mdr",
	"men": "men", "mga": "mga", "mic": "mic", "min": "min", "mis": "mis",
	"mkd": "mkd", "mkh": "mkh", "mlg": "mlg", "mlt": "mlt", "mnc": "mnc",
	"mni": "mni", "mno": "mno", "moh": "moh", "mol": "mol", "mon": "mon",
	"mos": "mos", "mri": "mri", "msa": "msa", "mul": "mul", "mun": "mun",
	"mus": "mus", "mwl": "mwl", "mwr": "mwr", "myn": "myn", "myv": "myv",
	"nah": "nah", "nai": "nai", "nap": "nap", "nau": "nau", "nav": "nav",
	"nbl": "nbl", "nde": "nde", "ndo": "ndo", "nds": "nds", "nep": "nep",
	"new": "new", "nia": "nia", "nic": "nic", "niu": "niu", "nno": "nno",
	"nob": "nob", "nog": "nog", "non": "non", "nor": "nor", "nso": "nso",
	"nub": "nub", "nwc": "nwc", "nya": "nya", "nym": "nym", "nyn": "nyn",
	"nyo": "nyo", "nzi": "nzi", "oci": "oci", "oji": "oji", "ori": "ori",
	"orm": "orm", "osa": "osa", "oss": "oss", "ota": "ota", "oto": "oto",
	"paa": "paa", "pag": "pag", "pal": "pal", "pam": "pam", "pan": "pan",
	"pap": "pap", "pau": "pau", "peo": "peo", "per": "per", "phi": "phi",
	"phn": "phn", "pli": "pli", "pol": "pol", "pon": "pon", "por": "por",
	"pra": "pra", "pro": "pro", "pus": "pus", "que": "que", "raj": "raj",
	"rap": "rap", "rar": "rar", "roa": "roa", "roh": "roh", "rom": "rom",
	"rum": "rum", "run": "run", "rus": "rus", "sad": "sad", "sag": "sag",
	"sah": "sah", "sai": "sai", "sal": "sal", "sam": "sam", "san": "san",
	"sas": "sas", "sat": "sat", "scn": "scn", "sco": "sco", "sel": "sel",
	"sem": "sem", "sga": "sga", "sgn": "sgn", "shn": "shn", "sid": "sid",
	"sin": "sin", "sio": "sio", "sit": "sit", "sla": "sla", "slo": "slo",
	"slv": "slv", "sma": "sma", "sme": "sme", "smi": "smi", "smj": "smj",
	"smn": "smn", "smo": "smo", "sms": "sms", "sna": "sna", "snd": "snd",
	"snk": "snk", "sog": "sog", "som": "som", "son": "son", "sot": "sot",
	"spa": "spa", "srd": "srd", "srr": "srr", "ssa": "ssa", "ssw": "ssw",
	"suk": "suk", "sun": "sun", "sus": "sus", "sux": "sux", "swa": "swa",
	"swe": "swe", "syr": "syr", "tah": "tah", "tai": "tai", "tam": "tam",
	"tat": "tat", "tel": "tel", "tem": "tem", "ter": "ter", "tet": "tet",
	"tgk": "tgk", "tgl": "tgl", "tha": "tha", "tib": "tib", "tig": "tig",
	"tir": "tir", "tiv": "tiv", "tkl": "tkl", "tli": "tli", "tmh": "tmh",
	"tog": "tog", "ton": "ton", "tpi": "tpi", "tsi": "tsi", "tsn": "tsn",
	"tso": "tso", "tuk": "tuk", "tum": "tum", "tup": "tup", "tur": "tur",
	"tut": "tut", "tvl": "tvl", "twi": "twi", "tyv": "tyv", "udm": "udm",
	"uga": "uga", "uig": "uig", "ukr": "ukr", "umb": "umb", "und": "und",
	"urd": "urd", "uzb": "uzb", "vai": "vai", "ven": "ven", "vie": "vie",
	"vol": "vol", "vot": "vot", "wak": "wak", "wal": "wal", "war": "war",
	"was": "was", "wel": "wel", "wen": "wen", "wln": "wln", "wol": "wol",
	"xho": "xho", "yao": "yao", "yap": "yap", "yid": "yid", "yor": "yor",
	"zap": "zap", "zen": "zen", "zha": "zha", "znd": "znd", "zul": "zul",
	"zun": "zun",
}

// iso6391To6392 maps ISO 639-1 two-letter codes to ISO 639-2/B three-letter codes.
var iso6391To6392 = map[string]string{
	"aa": "aar", "ab": "abk", "ae": "ave", "af": "afr", "ak": "aka",
	"am": "amh", "an": "arg", "ar": "ara", "as": "asm", "av": "ava",
	"ay": "aym", "az": "aze", "ba": "bak", "be": "bel", "bg": "bul",
	"bh": "bih", "bi": "bis", "bm": "bam", "bn": "ben", "bo": "bod",
	"br": "bre", "bs": "bos", "ca": "cat", "ce": "che", "ch": "cha",
	"co": "cos", "cr": "cre", "cs": "ces", "cu": "chu", "cv": "chv",
	"cy": "cym", "da": "dan", "de": "deu", "dv": "div", "dz": "dzo",
	"ee": "ewe", "el": "ell", "en": "eng", "eo": "epo", "es": "spa",
	"et": "est", "eu": "eus", "fa": "fas", "ff": "ful", "fi": "fin",
	"fj": "fij", "fo": "fao", "fr": "fra", "fy": "fry", "ga": "gle",
	"gd": "gla", "gl": "glg", "gn": "grn", "gu": "guj", "gv": "glv",
	"ha": "hau", "he": "heb", "hi": "hin", "ho": "hmo", "hr": "hrv",
	"ht": "hat", "hu": "hun", "hy": "hye", "hz": "her", "ia": "ina",
	"id": "ind", "ie": "ile", "ig": "ibo", "ii": "iii", "ik": "ipk",
	"io": "ido", "is": "isl", "it": "ita", "iu": "iku", "ja": "jpn",
	"jv": "jav", "ka": "kat", "kg": "kon", "ki": "kik", "kj": "kua",
	"kk": "kaz", "kl": "kal", "km": "khm", "kn": "kan", "ko": "kor",
	"kr": "kau", "ks": "kas", "ku": "kur", "kv": "kom", "kw": "cor",
	"ky": "kir", "la": "lat", "lb": "ltz", "lg": "lug", "li": "lim",
	"ln": "lin", "lo": "lao", "lt": "lit", "lu": "lub", "lv": "lav",
	"mg": "mlg", "mh": "mah", "mi": "mri", "mk": "mkd", "ml": "mal",
	"mn": "mon", "mr": "mar", "ms": "msa", "mt": "mlt", "my": "mya",
	"na": "nau", "nb": "nob", "nd": "nde", "ne": "nep", "ng": "ndo",
	"nl": "nld", "nn": "nno", "no": "nor", "nr": "nbl", "nv": "nav",
	"ny": "nya", "oc": "oci", "oj": "oji", "om": "orm", "or": "ori",
	"os": "oss", "pa": "pan", "pi": "pli", "pl": "pol", "ps": "pus",
	"pt": "por", "qu": "que", "rm": "roh", "rn": "run", "ro": "ron",
	"ru": "rus", "rw": "kin", "sa": "san", "sc": "srd", "sd": "snd",
	"se": "sme", "sg": "sag", "si": "sin", "sk": "slk", "sl": "slv",
	"sm": "smo", "sn": "sna", "so": "som", "sq": "sqi", "sr": "srp",
	"ss": "ssw", "st": "sot", "su": "sun", "sv": "swe", "sw": "swa",
	"ta": "tam", "te": "tel", "tg": "tgk", "th": "tha", "ti": "tir",
	"tk": "tuk", "tl": "tgl", "tn": "tsn", "to": "ton", "tr": "tur",
	"ts": "tso", "tt": "tat", "tw": "twi", "ty": "tyv", "ug": "uig",
	"uk": "ukr", "ur": "urd", "uz": "uzb", "ve": "ven", "vi": "vie",
	"vo": "vol", "wa": "wln", "wo": "wol", "xh": "xho", "yi": "yid",
	"yo": "yor", "za": "zha", "zh": "zho", "zu": "zul",
}

// languageTokenPatterns defines regex patterns for extracting language codes from filenames.
// The language code is stripped from the filename to avoid it contaminating the movie title.
var languageTokenPatterns = []*regexp.Regexp{
	// Bracketed: [eng], {eng}, [fi], {fi}
	regexp.MustCompile(`(?i)[\[{]([a-z]{2,3})[\}\]]`),
	// After year/dot separator: Movie.2020.eng.mkv, Movie.2020.fi.mkv
	regexp.MustCompile(`(?i)(?:^|[.\s_-])([a-z]{2,3})(?:$|[.\s_-])`),
}

// extractLanguage checks for known ISO 639-2/B language codes in the filename.
// Returns the three-letter code (lowercase) or empty string if not found.
func extractLanguage(filename string) string {
	filenameLower := strings.ToLower(filename)

	// First pass: look for bracketed language codes (e.g., [eng], {fre}, [fi])
	// These are the most explicit indicator and take priority.
	for _, pat := range languageTokenPatterns[:1] {
		matches := pat.FindAllStringSubmatch(filenameLower, -1)
		for _, m := range matches {
			if len(m) >= 2 {
				code := m[1]
				if len(code) == 2 {
					if mapped, ok := iso6391To6392[code]; ok {
						return mapped
					}
				} else if _, ok := iso6392Codes[code]; ok {
					return code
				}
			}
		}
	}

	// Second pass: split on non-alpha characters and check each token
	// This handles patterns like Movie.2020.eng.mkv, Movie_2020_fre.mkv, Movie.2020.fi.mkv
	tokens := regexp.MustCompile(`[.\s_[\]{}()-]+`).Split(filenameLower, -1)
	for _, token := range tokens {
		if (len(token) == 2 || len(token) == 3) && token[0] >= 'a' && token[0] <= 'z' {
			var code string
			if len(token) == 2 {
				if mapped, ok := iso6391To6392[token]; ok {
					code = mapped
				} else {
					continue
				}
			} else {
				code = token
			}
			if _, ok := iso6392Codes[code]; ok {
				// Skip tokens that look like common non-language tokens
				if code == "the" || code == "and" || code == "dir" || code == "sub" ||
					code == "dvd" || code == "hdr" || code == "uhd" || code == "x20" ||
					code == "x26" || code == "vid" || code == "aud" || code == "cut" ||
					code == "pro" || code == "max" || code == "rem" || code == "web" ||
					code == "hdt" || code == "blu" || code == "ray" || code == "rip" ||
					code == "xvi" || code == "div" || code == "mp4" || code == "mkv" ||
					code == "avi" || code == "m2t" || code == "ts" {
					continue
				}
				return code
			}
		}
	}

	return ""
}

// extractEdition extracts the movie edition from the filename
// Looks for edition markers like "Extended Cut", "Director's Cut", "Remastered", "Proper", etc.
func extractEdition(filename string) string {
	// Common edition patterns
	editionPatterns := []string{
		"Extended Cut",
		"Director's Cut",
		"Directors Cut",
		"Theatrical Cut",
		"Final Cut",
		"Remastered",
		"Restored",
		"Unrated",
		"Uncut",
		"Special Edition",
		"Limited Edition",
		"Criterion Collection",
		"Proper",
		"Hybrid",
	}

	filenameLower := strings.ToLower(filename)
	for _, edition := range editionPatterns {
		editionLower := strings.ToLower(edition)
		if strings.Contains(filenameLower, editionLower) {
			return edition
		}
	}

	return ""
}
