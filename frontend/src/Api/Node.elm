module Api.Node exposing
    ( Node
    , delete
    , description
    , get
    , getCmd
    , insert
    , list
    , postCmd
    , postGroups
    , postPoints
    , sysStateOffline
    , sysStateOnline
    , sysStatePowerOff
    , typeDevice
    , typeGroup
    , typeUser
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


typeDevice : String
typeDevice =
    "device"


typeGroup : String
typeGroup =
    "group"


typeUser : String
typeUser =
    "user"


type alias Node =
    { id : String
    , typ : String
    , points : List Point
    , groups : List String
    }


type alias NodeCmd =
    { cmd : String
    , detail : String
    }


decodeList : Decode.Decoder (List Node)
decodeList =
    Decode.list decode


decode : Decode.Decoder Node
decode =
    Decode.succeed Node
        |> required "id" Decode.string
        |> required "type" Decode.string
        |> optional "points" (Decode.list Point.decode) []
        |> optional "groups" (Decode.list Decode.string) []


decodeCmd : Decode.Decoder NodeCmd
decodeCmd =
    Decode.succeed NodeCmd
        |> required "cmd" Decode.string
        |> optional "detail" Decode.string ""


encode : Node -> Encode.Value
encode node =
    Encode.object
        [ ( "id", Encode.string node.id )
        , ( "type", Encode.string node.typ )
        ]


encodeGroups : List String -> Encode.Value
encodeGroups groups =
    Encode.list Encode.string groups


encodeNodeCmd : NodeCmd -> Encode.Value
encodeNodeCmd cmd =
    Encode.object
        [ ( "cmd", Encode.string cmd.cmd )
        , ( "detail", Encode.string cmd.detail )
        ]


description : Node -> String
description d =
    case Point.getPoint d.points "" Point.typeDescription 0 of
        Just point ->
            point.text

        Nothing ->
            ""


list :
    { token : String
    , onResponse : Data (List Node) -> msg
    }
    -> Cmd msg
list options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes" ] []
        , expect = Api.Data.expectJson options.onResponse decodeList
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


get :
    { token : String
    , id : String
    , onResponse : Data Node -> msg
    }
    -> Cmd msg
get options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id ] []
        , expect = Api.Data.expectJson options.onResponse decode
        , body = Http.emptyBody
        , timeout = Just <| 5 * 1000
        , tracker = Nothing
        }


getCmd :
    { token : String
    , id : String
    , onResponse : Data NodeCmd -> msg
    }
    -> Cmd msg
getCmd options =
    Http.request
        { method = "GET"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "cmd" ] []
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
        , url = Url.Builder.absolute [ "v1", "nodes", options.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = Http.emptyBody
        , timeout = Nothing
        , tracker = Nothing
        }


insert :
    { token : String
    , node : Node
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
insert options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.node.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.node |> encode |> Http.jsonBody
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
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "groups" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.groups |> encodeGroups |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postCmd :
    { token : String
    , id : String
    , cmd : NodeCmd
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
postCmd options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "cmd" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.cmd |> encodeNodeCmd |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


postPoints :
    { token : String
    , id : String
    , points : List Point
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
postPoints options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "points" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.points |> Point.encodeList |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
