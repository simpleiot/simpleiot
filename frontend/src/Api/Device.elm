module Api.Device exposing
    ( Device
    , delete
    , description
    , get
    , getCmd
    , list
    , postCmd
    , postGroups
    , postPoint
    , sysStateOffline
    , sysStateOnline
    , sysStatePowerOff
    )

import Api.Data exposing (Data)
import Api.Point as Point exposing (Point)
import Api.Response as Response exposing (Response)
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
    , points : List Point
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
        |> optional "points" (Decode.list Point.decode) []
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
    case Point.getPoint d.points "" Point.typeDescription 0 of
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


get :
    { token : String
    , id : String
    , onResponse : Data Device -> msg
    }
    -> Cmd msg
get options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices", options.id ] []
        , expect = Api.Data.expectJson options.onResponse decode
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


getCmd :
    { token : String
    , id : String
    , onResponse : Data DeviceCmd -> msg
    }
    -> Cmd msg
getCmd options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices", options.id, "cmd" ] []
        , expect = Api.Data.expectJson options.onResponse decodeCmd
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


delete :
    { token : String
    , id : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
delete options =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices", options.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


postGroups :
    { token : String
    , id : String
    , groups : List String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
postGroups options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices", options.id, "groups" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.groups |> encodeGroups |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postCmd :
    { token : String
    , id : String
    , cmd : DeviceCmd
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
postCmd options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices", options.id, "cmd" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.cmd |> encodeDeviceCmd |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postPoint :
    { token : String
    , id : String
    , point : Point
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
postPoint options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "devices", options.id, "points" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = [ options.point ] |> Point.encodeList |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
