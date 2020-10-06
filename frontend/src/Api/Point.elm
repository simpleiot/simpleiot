module Api.Point exposing
    ( Point
    , decode
    , encode
    , encodeList
    , getLatest
    , getPoint
    , newText
    , newValue
    , renderPoint
    , typeAppVersion
    , typeCmdPending
    , typeDescription
    , typeHwVersion
    , typeOSVersion
    , typeStartApp
    , typeStartSystem
    , typeSwUpdateError
    , typeSwUpdatePercComplete
    , typeSwUpdateRunning
    , typeSwUpdateState
    , typeSysState
    , typeUpdateApp
    , typeUpdateOS
    , updatePoint
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



-- Point should match data/Point.go


type alias Point =
    { id : String
    , typ : String
    , index : Int
    , time : Time.Posix
    , value : Float
    , text : String
    , min : Float
    , max : Float
    }


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


encode : Point -> Json.Encode.Value
encode s =
    Json.Encode.object
        [ ( "id", Json.Encode.string <| s.id )
        , ( "type", Json.Encode.string <| s.typ )
        , ( "index", Json.Encode.int <| s.index )
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
        |> optional "type" Decode.string ""
        |> optional "index" Decode.int 0
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


getPoint : List Point -> String -> String -> Int -> Maybe Point
getPoint points id typ index =
    List.Extra.find
        (\p ->
            id == p.id && typ == p.typ && index == p.index
        )
        points


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
