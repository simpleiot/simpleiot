module Api.Device exposing
    ( Device
    , description
    , list
    , sysStateOffline
    , sysStateOnline
    , sysStatePowerOff
    )

import Api.Data exposing (Data)
import Api.Point as P
import Http
import Json.Decode as Decode
import Json.Decode.Pipeline exposing (optional, required)
import Json.Encode as Encode
import Url.Builder


sysStatePowerOff : Int
sysStatePowerOff =
    1


sysStateOffline : Int
sysStateOffline =
    2


sysStateOnline : Int
sysStateOnline =
    3


type alias Device =
    { id : String
    , points : List P.Point
    , groups : List String
    }


type alias DeviceCmd =
    { cmd : String
    , detail : String
    }


decodeList : Decode.Decoder (List Device)
decodeList =
    Decode.list decode


decode : Decode.Decoder Device
decode =
    Decode.succeed Device
        |> required "id" Decode.string
        |> optional "points" (Decode.list P.decode) []
        |> optional "groups" (Decode.list Decode.string) []


decodeCmd : Decode.Decoder DeviceCmd
decodeCmd =
    Decode.succeed DeviceCmd
        |> required "cmd" Decode.string
        |> optional "detail" Decode.string ""


encodeGroups : List String -> Encode.Value
encodeGroups groups =
    Encode.list Encode.string groups


encodeDeviceCmd : DeviceCmd -> Encode.Value
encodeDeviceCmd cmd =
    Encode.object
        [ ( "cmd", Encode.string cmd.cmd )
        , ( "detail", Encode.string cmd.detail )
        ]


description : Device -> String
description d =
    case P.getPoint d.points "" P.typeDescription 0 of
        Just point ->
            point.text

        Nothing ->
            ""


list :
    { token : String
    , onResponse : Data (List Device) -> msg
    }
    -> Cmd msg
list options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices" ] []
        , expect = Api.Data.expectJson options.onResponse decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }
