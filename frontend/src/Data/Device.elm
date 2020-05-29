module Data.Device exposing
    ( Config
    , Device
    , DeviceCmd
    , decode
    , decodeList
    , encodeConfig
    , encodeDeviceCmd
    , encodeGroups
    )

--import Json.Encode as Encode

import Data.Sample as Sample
import Json.Decode as Decode
import Json.Decode.Extra
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode
import Time


type alias Device =
    { id : String
    , config : Config
    , state : State
    , groups : List String
    }


type alias Config =
    { description : String
    }


type alias State =
    { version : DeviceVersion
    , ios : List Sample.Sample
    , lastComm : Time.Posix
    }


type alias DeviceVersion =
    { os : String
    , app : String
    , hw : String
    }


type alias DeviceCmd =
    { id : String
    , cmd : String
    , detail : String
    }


emptyVersion : DeviceVersion
emptyVersion =
    DeviceVersion "" "" ""


decodeList : Decode.Decoder (List Device)
decodeList =
    Decode.list decode


decode : Decode.Decoder Device
decode =
    Decode.succeed Device
        |> required "id" Decode.string
        |> required "config" decodeConfig
        |> required "state" decodeState
        |> optional "groups" (Decode.list Decode.string) []


decodeConfig : Decode.Decoder Config
decodeConfig =
    Decode.succeed Config
        |> required "description" Decode.string


decodeState : Decode.Decoder State
decodeState =
    Decode.succeed State
        |> optional "version" decodeVersion emptyVersion
        |> optional "ios" (Decode.list Sample.decode) []
        |> optional "lastComm" Json.Decode.Extra.datetime (Time.millisToPosix 0)


decodeVersion : Decode.Decoder DeviceVersion
decodeVersion =
    Decode.succeed DeviceVersion
        |> required "os" Decode.string
        |> required "app" Decode.string
        |> required "hw" Decode.string


encodeConfig : Config -> Encode.Value
encodeConfig deviceConfig =
    Encode.object
        [ ( "description", Encode.string deviceConfig.description ) ]


encodeGroups : List String -> Encode.Value
encodeGroups groups =
    Encode.list Encode.string groups


encodeDeviceCmd : DeviceCmd -> Encode.Value
encodeDeviceCmd cmd =
    Encode.object
        [ ( "cmd", Encode.string cmd.cmd )
        , ( "detail", Encode.string cmd.detail )
        ]
