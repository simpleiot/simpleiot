module Api.Point exposing
    ( Point
    , blankMajicValue
    , clearText
    , decode
    , encodeList
    , filterDeleted
    , filterSpecialPoints
    , get
    , getAll
    , getBestDesc
    , getBool
    , getLatest
    , getText
    , getTextArray
    , getValue
    , input
    , keyHighRate
    , keyParent
    , keyPointKey
    , keyPointType
    , light
    , newText
    , renderPoint
    , renderPoint2
    , sort
    , switch
    , typeAction
    , typeActive
    , typeAddress
    , typeAuthToken
    , typeAutoDownload
    , typeAutoReboot
    , typeBatchPeriod
    , typeBaud
    , typeBinary
    , typeBitRate
    , typeBucket
    , typeChannel
    , typeClientServer
    , typeConditionType
    , typeConnected
    , typeControlled
    , typeData
    , typeDataFormat
    , typeDate
    , typeDebug
    , typeDebugPort
    , typeDefaultDialogs
    , typeDescription
    , typeDestination
    , typeDevice
    , typeDeviceID
    , typeDialogColor
    , typeDirectory
    , typeDisableSandbox
    , typeDisabled
    , typeDiscardDownload
    , typeDisplayCard
    , typeDownload
    , typeDownloadOS
    , typeEmail
    , typeEnd
    , typeError
    , typeErrorCount
    , typeErrorCountCRC
    , typeErrorCountCRCReset
    , typeErrorCountEOF
    , typeErrorCountEOFReset
    , typeErrorCountHR
    , typeErrorCountReset
    , typeErrorCountResetHR
    , typeExceptionURL
    , typeFallbackServer
    , typeFilePath
    , typeFirstName
    , typeFrequency
    , typeFrom
    , typeFullscreen
    , typeHRDest
    , typeHash
    , typeHrRx
    , typeHrRxReset
    , typeID
    , typeIP
    , typeIgnoreCertErr
    , typeIndex
    , typeInitialValue
    , typeKeyboardScale
    , typeLastName
    , typeLightSet
    , typeLog
    , typeMaxIncrement
    , typeMaxMessageLength
    , typeMaxValue
    , typeMinActive
    , typeMinIncrement
    , typeMinValue
    , typeModbusIOType
    , typeMsgsInDb
    , typeMsgsRecvdDb
    , typeMsgsRecvdDbReset
    , typeMsgsRecvdOther
    , typeMsgsRecvdOtherReset
    , typeName
    , typeNodeID
    , typeOSDownloaded
    , typeOSUpdate
    , typeOffline
    , typeOffset
    , typeOperator
    , typeOrg
    , typePass
    , typePeriod
    , typePhone
    , typePointKey
    , typePointType
    , typePollPeriod
    , typePort
    , typePrefix
    , typeProgress
    , typeProtocol
    , typeRate
    , typeRateHR
    , typeReadOnly
    , typeReboot
    , typeRefresh
    , typeRetryInterval
    , typeRotate
    , typeRoundTo
    , typeRx
    , typeRxReset
    , typeSID
    , typeSampleRate
    , typeScale
    , typeScreenResolution
    , typeServer
    , typeService
    , typeSignalType
    , typeSignalsInDb
    , typeSize
    , typeStart
    , typeSwitchSet
    , typeSyncCount
    , typeSyncCountReset
    , typeSyncParent
    , typeSysState
    , typeTag
    , typeTagPointType
    , typeTimeout
    , typeTombstone
    , typeTouchQuirk
    , typeTx
    , typeTxReset
    , typeType
    , typeURI
    , typeURL
    , typeUnits
    , typeValue
    , typeValueSet
    , typeValueText
    , typeValueType
    , typeVariableType
    , typeVersionApp
    , typeVersionHW
    , typeVersionOS
    , typeWeekday
      --  , keyNodeID
    , updatePoints
    , valueApp
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
    , valueNotify
    , valueNumber
    , valueOnOff
    , valuePlayAudio
    , valuePointValue
    , valueProcess
    , valueRTU
    , valueRandomWalk
    , valueSchedule
    , valueServer
    , valueSetValue
    , valueSine
    , valueSquare
    , valueSystem
    , valueTCP
    , valueText
    , valueTriangle
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


typeChannel : String
typeChannel =
    "channel"


typeDevice : String
typeDevice =
    "device"


typeDescription : String
typeDescription =
    "description"


typeFilePath : String
typeFilePath =
    "filePath"


typeDownload : String
typeDownload =
    "download"


typeProgress : String
typeProgress =
    "progress"


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


typeLightSet : String
typeLightSet =
    "lightSet"


typeSwitchSet : String
typeSwitchSet =
    "switchSet"


typeValueText : String
typeValueText =
    "valueText"


typeReadOnly : String
typeReadOnly =
    "readOnly"


typeSysState : String
typeSysState =
    "sysState"


typeVersionOS : String
typeVersionOS =
    "versionOS"


typeVersionApp : String
typeVersionApp =
    "versionApp"


typeVersionHW : String
typeVersionHW =
    "versionHW"


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


typeRx : String
typeRx =
    "rx"


typeHrRx : String
typeHrRx =
    "hrRx"


typeTx : String
typeTx =
    "tx"


typeRxReset : String
typeRxReset =
    "rxReset"


typeTxReset : String
typeTxReset =
    "txReset"


typeHrRxReset : String
typeHrRxReset =
    "hrRxReset"


typeBaud : String
typeBaud =
    "baud"


typeHRDest : String
typeHRDest =
    "hrDest"


typeMaxMessageLength : String
typeMaxMessageLength =
    "maxMessageLength"


typeID : String
typeID =
    "id"


typeLog : String
typeLog =
    "log"


typeAddress : String
typeAddress =
    "address"


typeError : String
typeError =
    "error"


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


typeSyncCount : String
typeSyncCount =
    "syncCount"


typeSyncCountReset : String
typeSyncCountReset =
    "syncCountReset"


typeErrorCountHR : String
typeErrorCountHR =
    "errorCountHR"


typeErrorCountResetHR : String
typeErrorCountResetHR =
    "errorCountResetHR"


typeMsgsInDb : String
typeMsgsInDb =
    "msgsInDb"


typeSignalsInDb : String
typeSignalsInDb =
    "signalsInDb"


typeMsgsRecvdDb : String
typeMsgsRecvdDb =
    "msgsRecvdDb"


typeMsgsRecvdOther : String
typeMsgsRecvdOther =
    "msgsRecvdOther"


typeMsgsRecvdDbReset : String
typeMsgsRecvdDbReset =
    "msgsRecvdDbReset"


typeMsgsRecvdOtherReset : String
typeMsgsRecvdOtherReset =
    "msgsRecvdOtherReset"


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


typeTimeout : String
typeTimeout =
    "timeout"


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


typePrefix : String
typePrefix =
    "prefix"


typeConditionType : String
typeConditionType =
    "conditionType"


valuePointValue : String
valuePointValue =
    "pointValue"


valueSchedule : String
valueSchedule =
    "schedule"


typeValueType : String
typeValueType =
    "valueType"


typeStart : String
typeStart =
    "start"


typeEnd : String
typeEnd =
    "end"


typeWeekday : String
typeWeekday =
    "weekday"


typeDate : String
typeDate =
    "date"


typePointType : String
typePointType =
    "pointType"


typePointKey : String
typePointKey =
    "pointKey"


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


valueContains : String
valueContains =
    "contains"


typeMinActive : String
typeMinActive =
    "minActive"


typeAction : String
typeAction =
    "action"


valueNotify : String
valueNotify =
    "notify"


valueSetValue : String
valueSetValue =
    "setValue"


valuePlayAudio : String
valuePlayAudio =
    "playAudio"


typeURL : String
typeURL =
    "url"


typeRotate : String
typeRotate =
    "rotate"


typeKeyboardScale : String
typeKeyboardScale =
    "keyboardscale"


typeFullscreen : String
typeFullscreen =
    "fullscreen"


typeDefaultDialogs : String
typeDefaultDialogs =
    "defaultdialogs"


typeDialogColor : String
typeDialogColor =
    "dialogcolor"


typeTouchQuirk : String
typeTouchQuirk =
    "touchquirk"


typeRetryInterval : String
typeRetryInterval =
    "retryinterval"


typeExceptionURL : String
typeExceptionURL =
    "exceptionurl"


typeIgnoreCertErr : String
typeIgnoreCertErr =
    "ignorecerterr"


typeDisableSandbox : String
typeDisableSandbox =
    "disablesandbox"


typeDebugPort : String
typeDebugPort =
    "debugport"


typeScreenResolution : String
typeScreenResolution =
    "screenresolution"


typeDisplayCard : String
typeDisplayCard =
    "displaycard"


typeService : String
typeService =
    "service"


valueTwilio : String
valueTwilio =
    "twilio"


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


typeNodeID : String
typeNodeID =
    "nodeID"


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


typeDisabled : String
typeDisabled =
    "disabled"


typeConnected : String
typeConnected =
    "connected"


typeControlled : String
typeControlled =
    "controlled"


typeOffline : String
typeOffline =
    "offline"


typeSignalType : String
typeSignalType =
    "signalType"


typeBatchPeriod : String
typeBatchPeriod =
    "batchPeriod"


typeIndex : String
typeIndex =
    "index"


typeFrequency : String
typeFrequency =
    "frequency"


typeMinValue : String
typeMinValue =
    "minValue"


typeMaxValue : String
typeMaxValue =
    "maxValue"


typeInitialValue : String
typeInitialValue =
    "initialValue"


typeMinIncrement : String
typeMinIncrement =
    "minIncrement"


typeMaxIncrement : String
typeMaxIncrement =
    "maxIncrement"


typeRoundTo : String
typeRoundTo =
    "roundTo"


typeSampleRate : String
typeSampleRate =
    "sampleRate"


typePeriod : String
typePeriod =
    "period"


typeName : String
typeName =
    "name"


typeData : String
typeData =
    "data"


typeSize : String
typeSize =
    "size"


typeHash : String
typeHash =
    "hash"


typeBitRate : String
typeBitRate =
    "bitRate"


typeRate : String
typeRate =
    "rate"


typeRateHR : String
typeRateHR =
    "rateHR"


valueSine : String
valueSine =
    "sine"


valueSquare : String
valueSquare =
    "square"


valueTriangle : String
valueTriangle =
    "triangle"


valueRandomWalk : String
valueRandomWalk =
    "random walk"


typeType : String
typeType =
    "type"


typeIP : String
typeIP =
    "ip"


typeDeviceID : String
typeDeviceID =
    "deviceID"


valueApp : String
valueApp =
    "app"


valueProcess : String
valueProcess =
    "process"


valueSystem : String
valueSystem =
    "system"


switch : String
switch =
    "switch"


input : String
input =
    "input"


light : String
light =
    "light"


typeServer : String
typeServer =
    "server"


typeFallbackServer : String
typeFallbackServer =
    "fallbackServer"


typeSyncParent : String
typeSyncParent =
    "syncParent"


typeDestination : String
typeDestination =
    "destination"


typeTagPointType : String
typeTagPointType =
    "tagPointType"


typeTag : String
typeTag =
    "tag"



--keyNodeID : String
--keyNodeID =
--    "nodeID"


keyParent : String
keyParent =
    "parent"


keyHighRate : String
keyHighRate =
    "highRate"


keyPointType : String
keyPointType =
    "pointType"


keyPointKey : String
keyPointKey =
    "pointKey"


typeDownloadOS : String
typeDownloadOS =
    "downloadOS"


typeReboot : String
typeReboot =
    "reboot"


typeAutoReboot : String
typeAutoReboot =
    "autoReboot"


typeAutoDownload : String
typeAutoDownload =
    "autoDownload"


typeOSDownloaded : String
typeOSDownloaded =
    "osDownloaded"


typeDiscardDownload : String
typeDiscardDownload =
    "discardDownload"


typeOSUpdate : String
typeOSUpdate =
    "osUpdate"


typeDirectory : String
typeDirectory =
    "directory"


typeRefresh : String
typeRefresh =
    "refresh"


typeBinary : String
typeBinary =
    "binary"



-- Point should match data/Point.go


type alias Point =
    { typ : String
    , key : String
    , time : Time.Posix
    , value : Float
    , text : String
    , tombstone : Int
    }


newText : String -> String -> String -> Point
newText typ key text =
    { typ = typ
    , key = key
    , time = Time.millisToPosix 0
    , value = 0
    , text = text
    , tombstone = 0
    }


specialPoints : List String
specialPoints =
    [ typeDescription
    , typeVersionHW
    , typeVersionOS
    , typeVersionApp
    , typeBaud
    , typeHRDest
    , typeMaxMessageLength
    , typeDebug
    , typeDisabled
    , typeConnected
    , typeControlled
    , typeOffline
    , typeError
    , typeErrorCount
    , typeErrorCountReset
    , typeSyncCount
    , typeSyncCountReset
    , typeErrorCountHR
    , typeErrorCountResetHR
    , typeLog
    , typePort
    , typeRx
    , typeHrRx
    , typeHrRxReset
    , typeRxReset
    , typeTx
    , typeTxReset
    , typeRate
    , typeRateHR
    , typeType
    , typeIP
    , typePeriod
    , typeName
    , typeAuthToken
    , typeValue
    , typeValueSet
    , typeLightSet
    , typeSwitchSet
    , typeDeviceID
    , switch
    , input
    , light
    , typeDirectory
    , typeRefresh
    , typeBinary
    ]


filterSpecialPoints : List Point -> List Point
filterSpecialPoints points =
    List.filter (\p -> not <| List.member p.typ specialPoints) points


filterDeleted : List Point -> List Point
filterDeleted points =
    List.filter (\p -> p.tombstone == 0) points


encode : Point -> Json.Encode.Value
encode p =
    Json.Encode.object
        [ ( "type", Json.Encode.string <| p.typ )
        , ( "key", Json.Encode.string <| p.key )
        , ( "time", Iso8601.encode <| p.time )
        , ( "value", Json.Encode.float <| p.value )
        , ( "text", Json.Encode.string <| p.text )
        , ( "tombstone", Json.Encode.int <| p.tombstone )
        ]


encodeList : List Point -> Json.Encode.Value
encodeList p =
    Json.Encode.list encode p


decode : Decode.Decoder Point
decode =
    Decode.succeed Point
        |> optional "type" Decode.string ""
        |> optional "key" Decode.string ""
        |> optional "time" Json.Decode.Extra.datetime (Time.millisToPosix 0)
        |> optional "value" Decode.float 0
        |> optional "text" Decode.string ""
        |> optional "tombstone" Decode.int 0


renderPoint : Point -> String
renderPoint s =
    let
        key =
            if s.key == "" then
                ""

            else
                s.key ++ ":"

        value =
            if s.text /= "" then
                s.text

            else
                Round.round 2 s.value ++ ":"

        typ =
            s.typ ++ ":"
    in
    typ ++ key ++ " " ++ value


renderPoint2 : Point -> { desc : String, value : String }
renderPoint2 s =
    let
        key =
            if s.key == "" then
                ""

            else
                s.key ++ ":"

        value =
            if s.text /= "" then
                s.text

            else
                Round.round 2 s.value
    in
    { desc = s.typ ++ ":" ++ key, value = value }


updatePoint : List Point -> Point -> List Point
updatePoint points point =
    case
        List.Extra.findIndex
            (\p ->
                point.typ == p.typ && point.key == p.key
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


get : List Point -> String -> String -> Maybe Point
get points typ key =
    let
        keyS =
            if key == "" then
                "0"

            else
                key
    in
    List.Extra.find
        (\p ->
            typ == p.typ && keyS == p.key
        )
        points


getAll : List Point -> String -> List Point
getAll points typ =
    List.filter (\p -> typ == p.typ) points


getText : List Point -> String -> String -> String
getText points typ key =
    case
        get points typ key
    of
        Just found ->
            found.text

        Nothing ->
            ""


getTextArray : List Point -> String -> List String
getTextArray points typ =
    List.map .text <|
        List.sortWith
            (\a b ->
                let
                    aInt =
                        Maybe.withDefault 0 (String.toInt a.key)

                    bInt =
                        Maybe.withDefault 0 (String.toInt b.key)
                in
                compare aInt bInt
            )
        <|
            List.foldl
                (\p acc ->
                    if p.typ == typ && p.tombstone == 0 then
                        p :: acc

                    else
                        acc
                )
                []
                points


getBestDesc : List Point -> String
getBestDesc points =
    let
        firstName =
            getText points typeFirstName ""
    in
    if firstName /= "" then
        firstName ++ " " ++ getText points typeLastName ""

    else
        let
            desc =
                getText points typeDescription ""
        in
        if desc /= "" then
            desc

        else
            "no description"


getValue : List Point -> String -> String -> Float
getValue points typ key =
    case
        get points typ key
    of
        Just found ->
            found.value

        Nothing ->
            0


getBool : List Point -> String -> String -> Bool
getBool points typ key =
    getValue points typ key == 1


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
-- the text value is used by the form when editing things like decimal points


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


sort : Point -> Point -> Order
sort a b =
    if a.typ /= b.typ then
        compare a.typ b.typ

    else
        let
            keysAreInt =
                String.toInt a.key /= Nothing && String.toInt b.key /= Nothing

            aKeyInt =
                Maybe.withDefault 0 (String.toInt a.key)

            bKeyInt =
                Maybe.withDefault 0 (String.toInt b.key)
        in
        if keysAreInt && aKeyInt /= bKeyInt then
            compare aKeyInt bKeyInt

        else if a.key /= b.key then
            compare a.key b.key

        else
            compare a.value b.value
