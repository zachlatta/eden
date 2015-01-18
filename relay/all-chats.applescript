-- JSON Encoding From https://github.com/mgax/applescript-json

on encode(value)
	set type to class of value
	if type = integer or type = boolean then
		return value as text
	else if type = text then
		return encodeString(value)
	else if type = list then
		return encodeList(value)
	else if type = script then
		return value's toJson()
	else
		error "Unknown type " & type
	end if
end encode


on encodeList(value_list)
	set out_list to {}
	repeat with value in value_list
		copy encode(value) to end of out_list
	end repeat
	return "[" & join(out_list, ", ") & "]"
end encodeList


on encodeString(value)
	set rv to ""
	repeat with ch in value
		if id of ch = 34 then
			set quoted_ch to "\\\""
		else if id of ch = 92 then
			set quoted_ch to "\\\\"
		else if id of ch ≥ 32 and id of ch < 127 then
			set quoted_ch to ch
		else
			set quoted_ch to "\\u" & hex4(id of ch)
		end if
		set rv to rv & quoted_ch
	end repeat
	return "\"" & rv & "\""
end encodeString


on join(value_list, delimiter)
	set original_delimiter to AppleScript's text item delimiters
	set AppleScript's text item delimiters to delimiter
	set rv to value_list as text
	set AppleScript's text item delimiters to original_delimiter
	return rv
end join


on hex4(n)
	set digit_list to "0123456789abcdef"
	set rv to ""
	repeat until length of rv = 4
		set digit to (n mod 16)
		set n to (n - digit) / 16 as integer
		set rv to (character (1 + digit) of digit_list) & rv
	end repeat
	return rv
end hex4


on createDictWith(item_pairs)
	set item_list to {}
	
	script Dict
		on setkv(key, value)
			copy {key, value} to end of item_list
		end setkv
		
		on toJson()
			set item_strings to {}
			repeat with kv in item_list
				set key_str to encodeString(item 1 of kv)
				set value_str to encode(item 2 of kv)
				copy key_str & ": " & value_str to end of item_strings
			end repeat
			return "{" & join(item_strings, ", ") & "}"
		end toJson
	end script
	
	repeat with pair in item_pairs
		try
			Dict's setkv(item 1 of pair, item 2 of pair)
		end try
	end repeat
	
	return Dict
end createDictWith


on createDict()
	return createDictWith({})
end createDict

tell application "Messages"
	set textChats to text chats
end tell

set chatsList to []
repeat with aChat in textChats
	#log (id of aChat) as text
	set chatObj to createDict()
	
	set userList to []
	tell application "Messages"
		
		set chatSubject to (subject of aChat)
		if chatSubject is missing value then
			set chatSubject to ""
		end if
		
		set chatId to (id of aChat)
		repeat with aParticipant in (get participants of aChat)
			set userData to {{"first_name", (first name of aParticipant as text)}, {"last_name", (last name of aParticipant as string)}, {"handle", (handle of aParticipant as string)}}
			set userData to "{\"first_name\":\"" & (first name of aParticipant) & "\", \"last_name\":\"" & (last name of aParticipant) & "\", \"handle\":\"" & (handle of aParticipant) & "\"}"
			set userList to userList & userData
		end repeat
		
	end tell
	
	chatObj's setkv("participants", userList)
	chatObj's setkv("id", chatId)
	chatObj's setkv("first_message", chatSubject)
	set chatsList to chatsList & chatObj
	#log encode(chatObj)
end repeat

do shell script "echo " & quoted form of encode(chatsList)

