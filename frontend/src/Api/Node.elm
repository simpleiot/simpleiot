module Api.Node exposing
    ( Node
    , NodeView
    , Notification
    , copy
    , delete
    , description
    , getBestDesc
    , insert
    , list
    , move
    , notify
    , postPoints
    , typeAction
    , typeActionInactive
    , typeCanBus
    , typeCondition
    , typeDb
    , typeDevice
    , typeFile
    , typeGroup
    , typeMetrics
    , typeModbus
    , typeModbusIO
    , typeMsgService
    , typeNTP
    , typeNetworkManager
    , typeNetworkManagerConn
    , typeNetworkManagerDevice
    , typeOneWire
    , typeParticle
    , typeRule
    , typeSerialDev
    , typeShelly
    , typeShellyIO
    , typeSignalGenerator
    , typeSync
    , typeUpdate
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


typeParticle : String
typeParticle =
    "particle"


typeShelly : String
typeShelly =
    "shelly"


typeShellyIO : String
typeShellyIO =
    "shellyIo"


typeModbus : String
typeModbus =
    "modbus"


typeModbusIO : String
typeModbusIO =
    "modbusIo"


typeOneWire : String
typeOneWire =
    "oneWire"


typeSerialDev : String
typeSerialDev =
    "serialDev"


typeCanBus : String
typeCanBus =
    "canBus"


typeVariable : String
typeVariable =
    "variable"


typeSync : String
typeSync =
    "sync"


typeSignalGenerator : String
typeSignalGenerator =
    "signalGenerator"


typeFile : String
typeFile =
    "file"


typeMetrics : String
typeMetrics =
    "metrics"


typeNetworkManager : String
typeNetworkManager =
    "networkManager"


typeNetworkManagerDevice : String
typeNetworkManagerDevice =
    "networkManagerDevice"


typeNetworkManagerConn : String
typeNetworkManagerConn =
    "networkManagerConn"


typeNTP : String
typeNTP =
    "ntp"


typeUpdate : String
typeUpdate =
    "update"



-- Node corresponds with Go NodeEdge struct


type alias Node =
    { id : String
    , typ : String
    , hash : Int
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
        |> optional "hash" Decode.int 0
        |> required "parent" Decode.string
        |> optional "points" (Decode.list Point.decode) []
        |> optional "edgePoints" (Decode.list Point.decode) []


encode : Node -> Encode.Value
encode node =
    Encode.object
        [ ( "id", Encode.string node.id )
        , ( "type", Encode.string node.typ )
        , ( "hash", Encode.int node.hash )
        , ( "parent", Encode.string node.parent )
        , ( "points", Point.encodeList node.points )
        , ( "edgePoints", Point.encodeList node.edgePoints )
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
