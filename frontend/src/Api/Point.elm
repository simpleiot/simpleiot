module Api.Point exposing
    ( Point
    , blankMajicValue
    , clearText
    , decode
    , empty
    , encode
    , encodeList
    , filterSpecialPoints
    , get
    , getBestDesc
    , getBool
    , getLatest
    , getText
    , getValue
    , newText
    , newValue
    , renderPoint
    , typeActionType
    , typeActive
    , typeAddress
    , typeAppVersion
    , typeAuthToken
    , typeBaud
    , typeBucket
    , typeClientServer
    , typeCmdPending
    , typeDataFormat
    , typeDebug
    , typeDescription
    , typeEmail
    , typeErrorCount
    , typeErrorCountCRC
    , typeErrorCountCRCReset
    , typeErrorCountEOF
    , typeErrorCountEOFReset
    , typeErrorCountReset
    , typeFilePath
    , typeFirstName
    , typeFrom
    , typeHwVersion
    , typeID
    , typeLastName
    , typeMinActive
    , typeModbusIOType
    , typeNodeType
    , typeOSVersion
    , typeOffset
    , typeOperator
    , typeOrg
    , typePass
    , typePhone
    , typePointID
    , typePointIndex
    , typePointType
    , typePollPeriod
    , typePort
    , typeProtocol
    , typeReadOnly
    , typeSID
    , typeScale
    , typeService
    , typeStartApp
    , typeStartSystem
    , typeSwUpdateError
    , typeSwUpdatePercComplete
    , typeSwUpdateRunning
    , typeSwUpdateState
    , typeSysState
    , typeTombstone
    , typeURI
    , typeUnits
    , typeUpdateApp
    , typeUpdateOS
    , typeValue
    , typeValueSet
    , typeValueType
    , typeVariableType
    , updatePoint
    , updatePoints
    , valueActionNotify
    , valueActionPlayAudio
    , valueActionSetValue
    , valueActionSetValueBool
    , valueActionSetValueText
    , valueClient
    , valueContains
    , valueEqual
    , valueFLOAT32
    , valueGreaterThan
    , valueINT16
    , valueINT32
    , valueLessThan
    , valueModbusCoil
    , valueModbusDiscreteInput
    , valueModbusHoldingRegister
    , valueModbusInputRegister
    , valueNotEqual
    , valueNumber
    , valueOff
    , valueOn
    , valueOnOff
    , valueRTU
    , valueSMTP
    , valueServer
    , valueSysStateOffline
    , valueSysStateOnline
    , valueSysStatePowerOff
    , valueSysStateUnknown
    , valueTCP
    , valueText
    , valueTwilio
    , valueUINT16
    , valueUINT32
    )

import Iso8601
import Json.Decode as Decode
import Json.Decode.Extra
import Json.Decode.Pipeline exposing (optional)
import Json.Encode
import List.Extra
import Round
import Time


typeDescription : String
typeDescription =
    "description"


typeFilePath : String
typeFilePath =
    "filePath"


typeScale : String
typeScale =
    "scale"


typeOffset : String
typeOffset =
    "offset"


typeUnits : String
typeUnits =
    "units"


typeValue : String
typeValue =
    "value"


typeValueSet : String
typeValueSet =
    "valueSet"


typeReadOnly : String
typeReadOnly =
    "readOnly"


typeCmdPending : String
typeCmdPending =
    "cmdPending"


typeSwUpdateState : String
typeSwUpdateState =
    "swUpdateState"


typeStartApp : String
typeStartApp =
    "startApp"


typeStartSystem : String
typeStartSystem =
    "startSystem"


typeUpdateOS : String
typeUpdateOS =
    "updateOS"


typeUpdateApp : String
typeUpdateApp =
    "updateApp"


typeSysState : String
typeSysState =
    "sysState"


valueSysStateUnknown : String
valueSysStateUnknown =
    "unknown"


valueSysStatePowerOff : String
valueSysStatePowerOff =
    "powerOff"


valueSysStateOffline : String
valueSysStateOffline =
    "offline"


valueSysStateOnline : String
valueSysStateOnline =
    "online"


typeSwUpdateRunning : String
typeSwUpdateRunning =
    "swUpdateRunning"


typeSwUpdateError : String
typeSwUpdateError =
    "swUpdateError"


typeSwUpdatePercComplete : String
typeSwUpdatePercComplete =
    "swUpdatePercComplete"


typeOSVersion : String
typeOSVersion =
    "osVersion"


typeAppVersion : String
typeAppVersion =
    "appVersion"


typeHwVersion : String
typeHwVersion =
    "hwVersion"


typeFirstName : String
typeFirstName =
    "firstName"


typeLastName : String
typeLastName =
    "lastName"


typeEmail : String
typeEmail =
    "email"


typePhone : String
typePhone =
    "phone"


typePass : String
typePass =
    "pass"


typePort : String
typePort =
    "port"


typeBaud : String
typeBaud =
    "baud"


typeID : String
typeID =
    "id"


typeAddress : String
typeAddress =
    "address"


typeErrorCount : String
typeErrorCount =
    "errorCount"


typeErrorCountEOF : String
typeErrorCountEOF =
    "errorCountEOF"


typeErrorCountCRC : String
typeErrorCountCRC =
    "errorCountCRC"


typeErrorCountReset : String
typeErrorCountReset =
    "errorCountReset"


typeErrorCountEOFReset : String
typeErrorCountEOFReset =
    "errorCountEOFReset"


typeErrorCountCRCReset : String
typeErrorCountCRCReset =
    "errorCountCRCReset"


typeProtocol : String
typeProtocol =
    "protocol"


valueRTU : String
valueRTU =
    "RTU"


valueTCP : String
valueTCP =
    "TCP"


typeModbusIOType : String
typeModbusIOType =
    "modbusIoType"


valueModbusDiscreteInput : String
valueModbusDiscreteInput =
    "modbusDiscreteInput"


valueModbusCoil : String
valueModbusCoil =
    "modbusCoil"


valueModbusInputRegister : String
valueModbusInputRegister =
    "modbusInputRegister"


valueModbusHoldingRegister : String
valueModbusHoldingRegister =
    "modbusHoldingRegister"


typeDataFormat : String
typeDataFormat =
    "dataFormat"


typeDebug : String
typeDebug =
    "debug"


typePollPeriod : String
typePollPeriod =
    "pollPeriod"


valueUINT16 : String
valueUINT16 =
    "uint16"


valueINT16 : String
valueINT16 =
    "int16"


valueUINT32 : String
valueUINT32 =
    "uint32"


valueINT32 : String
valueINT32 =
    "int32"


valueFLOAT32 : String
valueFLOAT32 =
    "float32"


typeClientServer : String
typeClientServer =
    "clientServer"


valueClient : String
valueClient =
    "client"


valueServer : String
valueServer =
    "server"


typeURI : String
typeURI =
    "uri"


typeValueType : String
typeValueType =
    "valueType"


typePointID : String
typePointID =
    "pointID"


typePointType : String
typePointType =
    "pointType"


typePointIndex : String
typePointIndex =
    "pointIndex"


typeOperator : String
typeOperator =
    "operator"


valueGreaterThan : String
valueGreaterThan =
    ">"


valueLessThan : String
valueLessThan =
    "<"


valueEqual : String
valueEqual =
    "="


valueNotEqual : String
valueNotEqual =
    "!="


valueOn : String
valueOn =
    "on"


valueOff : String
valueOff =
    "off"


valueContains : String
valueContains =
    "contains"


typeMinActive : String
typeMinActive =
    "minActive"


typeActionType : String
typeActionType =
    "actionType"


valueActionNotify : String
valueActionNotify =
    "notify"


valueActionSetValue : String
valueActionSetValue =
    "setValue"


valueActionPlayAudio : String
valueActionPlayAudio =
    "playAudio"


valueActionSetValueBool : String
valueActionSetValueBool =
    "setValueBool"


valueActionSetValueText : String
valueActionSetValueText =
    "setValueText"


typeService : String
typeService =
    "service"


valueTwilio : String
valueTwilio =
    "twilio"


valueSMTP : String
valueSMTP =
    "smtp"


typeSID : String
typeSID =
    "sid"


typeAuthToken : String
typeAuthToken =
    "authToken"


typeFrom : String
typeFrom =
    "from"


typeVariableType : String
typeVariableType =
    "variableType"


typeNodeType : String
typeNodeType =
    "nodeType"


typeTombstone : String
typeTombstone =
    "tombstone"


valueOnOff : String
valueOnOff =
    "onOff"


valueNumber : String
valueNumber =
    "number"


valueText : String
valueText =
    "text"


typeActive : String
typeActive =
    "active"


typeBucket : String
typeBucket =
    "bucket"


typeOrg : String
typeOrg =
    "org"



-- Point should match data/Point.go


type alias Point =
    { id : String
    , index : Int
    , typ : String
    , time : Time.Posix
    , value : Float
    , text : String
    , min : Float
    , max : Float
    }


empty : Point
empty =
    Point
        ""
        0
        ""
        (Time.millisToPosix 0)
        0
        ""
        0
        0


newValue : String -> String -> Float -> Point
newValue id typ value =
    { id = id
    , typ = typ
    , index = 0
    , time = Time.millisToPosix 0
    , value = value
    , text = ""
    , min = 0
    , max = 0
    }


newText : String -> String -> String -> Point
newText id typ text =
    { id = id
    , typ = typ
    , index = 0
    , time = Time.millisToPosix 0
    , value = 0
    , text = text
    , min = 0
    , max = 0
    }


specialPoints : List String
specialPoints =
    [ typeDescription
    , typeHwVersion
    , typeOSVersion
    , typeAppVersion
    ]


filterSpecialPoints : List Point -> List Point
filterSpecialPoints points =
    List.filter (\p -> not <| List.member p.typ specialPoints) points


encode : Point -> Json.Encode.Value
encode s =
    Json.Encode.object
        [ ( "id", Json.Encode.string <| s.id )
        , ( "index", Json.Encode.int <| s.index )
        , ( "type", Json.Encode.string <| s.typ )
        , ( "time", Iso8601.encode <| s.time )
        , ( "value", Json.Encode.float <| s.value )
        , ( "text", Json.Encode.string <| s.text )
        , ( "min", Json.Encode.float <| s.min )
        , ( "max", Json.Encode.float <| s.max )
        ]


encodeList : List Point -> Json.Encode.Value
encodeList p =
    Json.Encode.list encode p


decode : Decode.Decoder Point
decode =
    Decode.succeed Point
        |> optional "id" Decode.string ""
        |> optional "index" Decode.int 0
        |> optional "type" Decode.string ""
        |> optional "time" Json.Decode.Extra.datetime (Time.millisToPosix 0)
        |> optional "value" Decode.float 0
        |> optional "text" Decode.string ""
        |> optional "min" Decode.float 0
        |> optional "max" Decode.float 0


renderPoint : Point -> String
renderPoint s =
    let
        id =
            if s.id == "" then
                ""

            else
                s.id ++ ": "

        value =
            if s.text /= "" then
                s.text

            else
                Round.round 2 s.value
    in
    id ++ value ++ " (" ++ s.typ ++ ")"


updatePoint : List Point -> Point -> List Point
updatePoint points point =
    case
        List.Extra.findIndex
            (\p ->
                point.id == p.id && point.typ == p.typ && point.index == p.index
            )
            points
    of
        Just index ->
            List.Extra.setAt index point points

        Nothing ->
            point :: points


updatePoints : List Point -> List Point -> List Point
updatePoints points newPoints =
    List.foldr
        (\newPoint updatedPoints -> updatePoint updatedPoints newPoint)
        points
        newPoints


get : List Point -> String -> Int -> String -> Maybe Point
get points id index typ =
    List.Extra.find
        (\p ->
            id == p.id && typ == p.typ && index == p.index
        )
        points


getText : List Point -> String -> Int -> String -> String
getText points id index typ =
    case
        get points id index typ
    of
        Just found ->
            found.text

        Nothing ->
            ""


getBestDesc : List Point -> String
getBestDesc points =
    let
        firstName =
            getText points "" 0 typeFirstName

        desc =
            getText points "" 0 typeDescription
    in
    if firstName /= "" then
        firstName ++ " " ++ getText points "" 0 typeLastName

    else if desc /= "" then
        desc

    else
        "no description"


getValue : List Point -> String -> Int -> String -> Float
getValue points id index typ =
    case
        get points id index typ
    of
        Just found ->
            found.value

        Nothing ->
            0


getBool : List Point -> String -> Int -> String -> Bool
getBool points id index typ =
    getValue points id index typ == 1


getLatest : List Point -> Maybe Point
getLatest points =
    List.foldl
        (\p result ->
            case result of
                Just point ->
                    if Time.posixToMillis p.time > Time.posixToMillis point.time then
                        Just p

                    else
                        Just point

                Nothing ->
                    Just p
        )
        Nothing
        points



-- clearText is used to sanitize points that have number values before saving.
-- the text value is used by the form when editting things like decimal points


blankMajicValue : String
blankMajicValue =
    "123BLANK123"


clearText : List Point -> List Point
clearText points =
    List.map
        (\p ->
            if p.value /= 0 || p.text == blankMajicValue then
                { p | text = "" }

            else
                p
        )
        points
