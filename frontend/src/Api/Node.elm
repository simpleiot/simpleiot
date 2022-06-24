module Api.Node exposing
    ( Node
    , NodeView
    , copy
    , delete
    , description
    , get
    , getBestDesc
    , getCmd
    , insert
    , list
    , move
    , notify
    , postCmd
    , postPoints
    , sysStateOffline
    , sysStateOnline
    , sysStatePowerOff
    , typeAction
    , typeActionInactive
    , typeCondition
    , typeDb
    , typeDevice
    , typeGroup
    , typeModbus
    , typeModbusIO
    , typeMsgService
    , typeOneWire
    , typeOneWireIO
    , typeRule
    , typeSerialDev
    , typeUpstream
    , typeUser
    , typeVariable
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


typeRule : String
typeRule =
    "rule"


typeCondition : String
typeCondition =
    "condition"


typeAction : String
typeAction =
    "action"


typeActionInactive : String
typeActionInactive =
    "actionInactive"


typeUser : String
typeUser =
    "user"


typeMsgService : String
typeMsgService =
    "msgService"


typeDb : String
typeDb =
    "db"


typeModbus : String
typeModbus =
    "modbus"


typeModbusIO : String
typeModbusIO =
    "modbusIo"


typeOneWire : String
typeOneWire =
    "oneWire"


typeOneWireIO : String
typeOneWireIO =
    "oneWireIO"


typeSerialDev : String
typeSerialDev =
    "serialDev"


typeVariable : String
typeVariable =
    "variable"


typeUpstream : String
typeUpstream =
    "upstream"



-- Node corresponds with Go NodeEdge struct


type alias Node =
    { id : String
    , typ : String
    , hash : String
    , parent : String
    , points : List Point
    , edgePoints : List Point
    }


type alias NodeView =
    { node : Node
    , feID : Int
    , parentID : String
    , hasChildren : Bool
    , expDetail : Bool
    , expChildren : Bool
    , mod : Bool
    }


type alias NodeCmd =
    { cmd : String
    , detail : String
    }


type alias NodeMove =
    { id : String
    , oldParent : String
    , newParent : String
    }


type alias NodeCopy =
    { id : String
    , newParent : String
    , duplicate : Bool
    }


type alias NodeDelete =
    { parent : String
    }


type alias Notification =
    { id : String
    , parent : String
    , sourceNode : String
    , subject : String
    , message : String
    }


decodeList : Decode.Decoder (List Node)
decodeList =
    Decode.list decode


decode : Decode.Decoder Node
decode =
    Decode.succeed Node
        |> required "id" Decode.string
        |> required "type" Decode.string
        |> optional "hash" Decode.string ""
        |> required "parent" Decode.string
        |> optional "points" (Decode.list Point.decode) []
        |> optional "edgePoints" (Decode.list Point.decode) []


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
        , ( "hash", Encode.string node.hash )
        , ( "parent", Encode.string node.parent )
        , ( "points", Point.encodeList node.points )
        , ( "edgePoints", Point.encodeList node.edgePoints )
        ]


encodeNodeCmd : NodeCmd -> Encode.Value
encodeNodeCmd cmd =
    Encode.object
        [ ( "cmd", Encode.string cmd.cmd )
        , ( "detail", Encode.string cmd.detail )
        ]


encodeNotification : Notification -> Encode.Value
encodeNotification not =
    Encode.object
        [ ( "id", Encode.string not.id )
        , ( "parent", Encode.string not.parent )
        , ( "sourceNode", Encode.string not.sourceNode )
        , ( "subject", Encode.string not.subject )
        , ( "message", Encode.string not.message )
        ]


encodeNodeMove : NodeMove -> Encode.Value
encodeNodeMove nodeMove =
    Encode.object
        [ ( "id", Encode.string nodeMove.id )
        , ( "oldParent", Encode.string nodeMove.oldParent )
        , ( "newParent", Encode.string nodeMove.newParent )
        ]


encodeNodeCopy : NodeCopy -> Encode.Value
encodeNodeCopy nodeCopy =
    Encode.object
        [ ( "id", Encode.string nodeCopy.id )
        , ( "newParent", Encode.string nodeCopy.newParent )
        , ( "duplicate", Encode.bool nodeCopy.duplicate )
        ]


encodeNodeDelete : NodeDelete -> Encode.Value
encodeNodeDelete nodeDelete =
    Encode.object
        [ ( "parent", Encode.string nodeDelete.parent )
        ]


description : Node -> String
description d =
    case Point.get d.points Point.typeDescription "" of
        Just point ->
            point.text

        Nothing ->
            ""


getBestDesc : Node -> String
getBestDesc n =
    Point.getBestDesc n.points


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
    , parent : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
delete options =
    Http.request
        { method = "DELETE"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = encodeNodeDelete { parent = options.parent } |> Http.jsonBody
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


notify :
    { token : String
    , not : Notification
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
notify options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.not.sourceNode, "not" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body = options.not |> encodeNotification |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


move :
    { token : String
    , id : String
    , oldParent : String
    , newParent : String
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
move options =
    Http.request
        { method = "POST"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "parents" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body =
            { id = options.id
            , oldParent = options.oldParent
            , newParent = options.newParent
            }
                |> encodeNodeMove
                |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }


copy :
    { token : String
    , id : String
    , newParent : String
    , duplicate : Bool
    , onResponse : Data Response -> msg
    }
    -> Cmd msg
copy options =
    Http.request
        { method = "PUT"
        , headers = [ Http.header "Authorization" <| "Bearer " ++ options.token ]
        , url = Url.Builder.absolute [ "v1", "nodes", options.id, "parents" ] []
        , expect = Api.Data.expectJson options.onResponse Response.decoder
        , body =
            { id = options.id
            , newParent = options.newParent
            , duplicate = options.duplicate
            }
                |> encodeNodeCopy
                |> Http.jsonBody
        , timeout = Nothing
        , tracker = Nothing
        }
