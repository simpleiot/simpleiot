module Pages.Home_ exposing (Model, Msg, NodeEdit, NodeMsg, NodeOperation, page)

import Api.Data as Data exposing (Data)
import Api.Nats as Nats
import Api.Node as Node exposing (Node, NodeView)
import Api.Point as Point exposing (Point)
import Api.Port as Port
import Api.Response exposing (Response)
import Auth
import Base64.Encode
import Components.NodeAction as NodeAction
import Components.NodeBrowser as NodeBrowser
import Components.NodeCanBus as NodeCanBus
import Components.NodeCondition as NodeCondition
import Components.NodeDb as NodeDb
import Components.NodeDevice as NodeDevice
import Components.NodeDeviceCred as NodeDeviceCred
import Components.NodeEnrollToken as NodeEnrollToken
import Components.NodeFile as File
import Components.NodeGpio as NodeGpio
import Components.NodeGps as NodeGps
import Components.NodeGroup as NodeGroup
import Components.NodeIio as NodeIio
import Components.NodeIioChannel as NodeIioChannel
import Components.NodeMessageService as NodeMessageService
import Components.NodeMetrics as NodeMetrics
import Components.NodeModbus as NodeModbus
import Components.NodeModbusIO as NodeModbusIO
import Components.NodeMqtt as NodeMqtt
import Components.NodeMqttDevice as NodeMqttDevice
import Components.NodeMqttSub as NodeMqttSub
import Components.NodeNTP as NodeNTP
import Components.NodeNetworkManager as NodeNetworkManager
import Components.NodeNetworkManagerConn as NodeNetworkManagerConn
import Components.NodeNetworkManagerDevice as NodeNetworkManagerDevice
import Components.NodeOneWire as NodeOneWire
import Components.NodeOneWireIO as NodeOneWireIO
import Components.NodeOptions exposing (CopyMove(..), GeneratedToken, findNode)
import Components.NodeParticle as NodeParticle
import Components.NodeProvisioning as NodeProvisioning
import Components.NodeProvisioningFile as NodeProvisioningFile
import Components.NodeRaw as NodeRaw
import Components.NodeRule as NodeRule
import Components.NodeSerialDev as NodeSerialDev
import Components.NodeShelly as NodeShelly
import Components.NodeShellyIO as NodeShellyIO
import Components.NodeSignalGenerator as SignalGenerator
import Components.NodeSparkplugDevice as NodeSparkplugDevice
import Components.NodeSparkplugGroup as NodeSparkplugGroup
import Components.NodeSparkplugNode as NodeSparkplugNode
import Components.NodeSync as NodeSync
import Components.NodeUpdate as NodeUpdate
import Components.NodeUser as NodeUser
import Components.NodeVariable as NodeVariable
import Effect exposing (Effect)
import Element exposing (..)
import Element.Background as Background
import Element.Font as Font
import Element.Input as Input
import File
import File.Select
import Gen.Params.Home_ exposing (Params)
import Http
import Json.Decode as Decode
import Page
import Request
import Shared
import Storage
import Task
import Time
import Tree exposing (Tree)
import UI.Button as Button
import UI.Form as Form
import UI.Icon as Icon
import UI.Layout
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)
import Utils.NodeTree as NodeTree
import View exposing (View)


page : Shared.Model -> Request.With Params -> Page.With Model Msg
page shared _ =
    Page.protected.advanced <|
        \user ->
            { init = init shared
            , update = update shared
            , view = view user shared
            , subscriptions = subscriptions
            }



-- INIT


type alias Model =
    { nodeEdit : Maybe NodeEdit
    , addPoint : { typ : String, key : String }
    , customNodeType : String
    , zone : Time.Zone
    , now : Time.Posix
    , nodes : List (Tree NodeView)
    , error : Maybe String
    , lastError : Time.Posix
    , nodeOp : NodeOperation
    , copyMove : CopyMove
    , scratch : String
    , nodeMsg : Maybe NodeMsg
    , token : String
    , generatedToken : Maybe GeneratedToken

    -- connected is whether the NATS connection is up; watching is the
    -- subject list last sent to it
    , connected : Bool
    , watching : List String
    }


type alias NodeMsg =
    { feID : Int
    , text : String
    , time : Time.Posix
    }


type NodeOperation
    = OpNone
    | OpNodeToAdd NodeToAdd
    | OpNodeMessage NodeMessage
    | OpNodeDelete Int String
    | OpNodePaste Int String


type alias NodeEdit =
    { feID : Int
    , points : List Point
    , viewRaw : Bool
    }


type alias NodeToAdd =
    { typ : Maybe String
    , feID : Int
    , parent : String
    }


type alias NodeMessage =
    { feID : Int
    , id : String
    , parent : String
    , message : String
    }


defaultModel : Model
defaultModel =
    Model
        Nothing
        { typ = "", key = "" }
        ""
        Time.utc
        (Time.millisToPosix 0)
        []
        Nothing
        (Time.millisToPosix 0)
        OpNone
        CopyMoveNone
        ""
        Nothing
        ""
        Nothing
        False
        []


init : Shared.Model -> ( Model, Effect Msg )
init shared =
    let
        token =
            case shared.storage.user of
                Just user ->
                    user.token

                Nothing ->
                    ""

        model =
            { defaultModel | token = token }
    in
    ( model
    , Effect.fromCmd <|
        Cmd.batch
            [ Task.perform Zone Time.here
            , Task.perform Tick Time.now
            , Nats.send (Nats.Connect token)
            ]
    )



-- UPDATE


type Msg
    = SignOut
    | Tick Time.Posix
    | Zone Time.Zone
    | EditNodePoint Int (List Point)
    | EditScratch String
    | UploadFile String Bool
    | GenerateKey String
    | ApiRespGenerateKey String (Data Node.GeneratedToken)
    | UploadSelected String Bool File.File
    | UploadContents String File.File String
    | ToggleExpChildren Int
    | ToggleExpDetail Int
    | DiscardNodeOp
    | DiscardEdits
    | AddNode Int String
    | MsgNode Int String String
    | PasteNode Int String
    | DeleteNode Int String
    | UpdateMsg String
    | SelectAddNodeType String
    | ApiDelete String String
    | ApiPostPoints String
    | ApiPostAddNode Int
    | ApiPostMoveNode Int String String String
    | ApiPutMirrorNode Int String String String
    | ApiPutDuplicateNode Int String String String
    | ApiPostNotificationNode
    | ApiRespDelete (Data Response)
    | ApiRespPostAddNode Int (Data Response)
    | ApiRespPostMoveNode Int (Data Response)
    | ApiRespPutMirrorNode Int (Data Response)
    | ApiRespPutDuplicateNode Int (Data Response)
    | ApiRespPostNotificationNode (Data Response)
    | CopyNode Int String String String
    | ClearClipboard
    | ToggleRaw Int
    | UpdateNewPointType String
    | UpdateNewPointKey String
    | UpdateCustomNodeType String
    | NatsEvent (Result Decode.Error Nats.Event)


{-| update runs the message and then, if what is on screen changed, tells
the connection which subjects to watch.
-}
update : Shared.Model -> Msg -> Model -> ( Model, Effect Msg )
update shared msg model =
    let
        ( new, effect ) =
            updateInner shared msg model

        subjects =
            NodeTree.watchSubjects new.nodes
    in
    if new.connected && subjects /= new.watching then
        ( { new | watching = subjects }
        , Effect.batch [ effect, Effect.fromCmd <| Nats.send (Nats.Watch subjects) ]
        )

    else
        ( new, effect )


updateInner : Shared.Model -> Msg -> Model -> ( Model, Effect Msg )
updateInner shared msg model =
    case msg of
        SignOut ->
            ( { model | connected = False }
            , Effect.batch
                [ Effect.fromCmd <| Nats.send Nats.Disconnect
                , Effect.fromCmd <| Storage.signOut shared.storage
                ]
            )

        EditNodePoint feID points ->
            let
                editPoints =
                    case model.nodeEdit of
                        Just ne ->
                            ne.points

                        Nothing ->
                            []

                viewRaw =
                    case model.nodeEdit of
                        Just ne ->
                            ne.viewRaw

                        Nothing ->
                            False
            in
            ( { model
                | nodeEdit =
                    Just
                        { feID = feID
                        , points = Point.updatePoints editPoints points
                        , viewRaw = viewRaw
                        }
                , scratch = ""
              }
            , Effect.none
            )

        EditScratch s ->
            ( { model | scratch = s }, Effect.none )

        UploadFile id binary ->
            ( model, Effect.fromCmd <| File.Select.file [ "" ] (UploadSelected id binary) )

        GenerateKey id ->
            ( model
            , Effect.fromCmd <|
                Node.generateKey
                    { token = model.token
                    , id = id
                    , onResponse = ApiRespGenerateKey id
                    }
            )

        ApiRespGenerateKey id resp ->
            case resp of
                Data.Success k ->
                    ( { model | generatedToken = Just { id = id, token = k.token } }
                    , Effect.none
                    )

                Data.Failure err ->
                    ( popError "Error generating token" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        UploadSelected id binary file ->
            let
                uploadContents =
                    UploadContents id file

                encode d =
                    Base64.Encode.encode (Base64.Encode.bytes d)

                task =
                    if binary then
                        Task.map encode (File.toBytes file)

                    else
                        File.toString file
            in
            -- File.toString results in Task x String, thus the complexity of one more step
            ( model, Effect.fromCmd <| Task.perform uploadContents task )

        UploadContents id file contents ->
            let
                pointName =
                    Point Point.typeName "0" model.now 3 0 (File.name file) 0

                pointData =
                    Point Point.typeData "0" model.now 3 0 contents 0
            in
            ( model, sendPoints model id [ pointName, pointData ] )

        ApiPostPoints id ->
            case model.nodeEdit of
                Just edit ->
                    let
                        points =
                            Point.clearText edit.points

                        -- optimistically update nodes
                        updatedNodes =
                            List.map
                                (Tree.map
                                    (\n ->
                                        if n.node.id == id then
                                            let
                                                node =
                                                    n.node
                                            in
                                            { n
                                                | node =
                                                    { node
                                                        | points = Point.updatePoints node.points points
                                                    }
                                            }

                                        else
                                            n
                                    )
                                )
                                model.nodes
                    in
                    ( { model | nodeEdit = Nothing, nodes = updatedNodes }
                    , sendPoints model id points
                    )

                Nothing ->
                    ( model, Effect.none )

        DiscardNodeOp ->
            ( { model | nodeOp = OpNone }, Effect.none )

        DiscardEdits ->
            ( { model | nodeEdit = Nothing }
            , Effect.none
            )

        ToggleExpChildren feID ->
            -- expanding fetches the children afresh, which doubles as a
            -- refresh; collapsing keeps what is there
            case NodeTree.findByFeID feID model.nodes of
                Just n ->
                    if n.expChildren then
                        ( { model | nodes = NodeTree.setExpanded feID False model.nodes }, Effect.none )

                    else
                        ( { model | nodes = NodeTree.setExpanded feID True model.nodes }
                        , fetchChildren n.anchor n.node.id
                        )

                Nothing ->
                    ( model, Effect.none )

        ToggleExpDetail feID ->
            let
                nodes =
                    toggleExpDetail model.nodes feID
            in
            ( { model | nodes = nodes }, Effect.none )

        AddNode feID id ->
            ( { model
                | nodeOp = OpNodeToAdd { typ = Nothing, feID = feID, parent = id }
              }
            , Effect.none
            )

        MsgNode feID id parent ->
            ( { model
                | nodeOp =
                    OpNodeMessage
                        { id = id
                        , feID = feID
                        , parent = parent
                        , message = ""
                        }
              }
            , Effect.none
            )

        PasteNode feID id ->
            ( { model | nodeOp = OpNodePaste feID id }, Effect.none )

        DeleteNode feID parent ->
            ( { model | nodeOp = OpNodeDelete feID parent }, Effect.none )

        UpdateMsg message ->
            case model.nodeOp of
                OpNodeMessage op ->
                    ( { model | nodeOp = OpNodeMessage { op | message = message } }, Effect.none )

                _ ->
                    ( model, Effect.none )

        SelectAddNodeType typ ->
            case model.nodeOp of
                OpNodeToAdd add ->
                    ( { model | nodeOp = OpNodeToAdd { add | typ = Just typ } }, Effect.none )

                _ ->
                    ( model, Effect.none )

        ApiPostAddNode parent ->
            -- FIXME optimistically update nodes
            case model.nodeOp of
                OpNodeToAdd addNode ->
                    case addNode.typ of
                        Just typ ->
                            ( { model | nodeOp = OpNone }
                            , Effect.fromCmd <|
                                Node.insert
                                    { token = model.token
                                    , onResponse = ApiRespPostAddNode parent
                                    , node =
                                        { id = ""
                                        , typ =
                                            if typ == "custom" then
                                                model.customNodeType

                                            else
                                                typ
                                        , hash = 0
                                        , parent = addNode.parent
                                        , points =
                                            [ Point.newText
                                                Point.typeDescription
                                                ""
                                                "New, please edit"
                                            ]
                                        , edgePoints = []
                                        }
                                    }
                            )

                        Nothing ->
                            ( { model | nodeOp = OpNone }, Effect.none )

                _ ->
                    ( { model | nodeOp = OpNone }, Effect.none )

        ApiPostMoveNode parent id src dest ->
            ( model
            , Effect.fromCmd <|
                Node.move
                    { token = model.token
                    , id = id
                    , oldParent = src
                    , newParent = dest
                    , onResponse = ApiRespPostMoveNode parent
                    }
            )

        ApiPutMirrorNode parent id src dest ->
            ( model
            , Effect.fromCmd <|
                Node.copy
                    { token = model.token
                    , id = id
                    , oldParent = src
                    , newParent = dest
                    , duplicate = False
                    , onResponse = ApiRespPutMirrorNode parent
                    }
            )

        ApiPutDuplicateNode parent id src dest ->
            ( model
            , Effect.fromCmd <|
                Node.copy
                    { token = model.token
                    , id = id
                    , oldParent = src
                    , newParent = dest
                    , duplicate = True
                    , onResponse = ApiRespPutDuplicateNode parent
                    }
            )

        ApiPostNotificationNode ->
            ( model
            , case model.nodeOp of
                OpNodeMessage msgNode ->
                    Effect.fromCmd <|
                        Node.notify
                            { token = model.token
                            , not =
                                { id = ""
                                , sourceNode = msgNode.id
                                , subject = ""
                                , message = msgNode.message
                                }
                            , onResponse = ApiRespPostNotificationNode
                            }

                _ ->
                    Effect.none
            )

        ApiDelete id parent ->
            -- optimistically update nodes
            let
                nodes =
                    -- FIXME Tree.filter (\d -> d.id /= id) model.nodes
                    model.nodes
            in
            ( { model | nodes = nodes, nodeOp = OpNone }
            , Effect.fromCmd <|
                Node.delete
                    { token = model.token
                    , id = id
                    , parent = parent
                    , onResponse = ApiRespDelete
                    }
            )

        Zone zone ->
            ( { model | zone = zone }, Effect.none )

        Tick now ->
            let
                nodeMsg =
                    Maybe.andThen
                        (\m ->
                            let
                                timeMs =
                                    Time.posixToMillis m.time

                                nowMs =
                                    Time.posixToMillis model.now
                            in
                            if nowMs - timeMs > 3000 then
                                Just m

                            else
                                Nothing
                        )
                        model.nodeMsg

                error =
                    if Time.posixToMillis now - Time.posixToMillis model.lastError > 5 * 1000 then
                        Nothing

                    else
                        model.error
            in
            ( { model | now = now, nodeMsg = nodeMsg, error = error }
            , Effect.none
            )

        ApiRespDelete resp ->
            -- the tombstone arrives as a live edge point and hides the node
            case resp of
                Data.Failure err ->
                    ( popError "Error deleting device" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        ApiRespPostAddNode parentFeID resp ->
            case resp of
                Data.Success _ ->
                    ( { model | nodes = List.map (expChildren parentFeID) model.nodes }
                    , refetchChildren model parentFeID
                    )

                Data.Failure err ->
                    ( popError "Error adding node" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        ApiRespPostMoveNode parent resp ->
            case resp of
                Data.Success _ ->
                    ( { model
                        | nodeOp = OpNone
                        , copyMove = CopyMoveNone
                        , nodes = List.map (expChildren parent) model.nodes
                      }
                    , refetchChildren model parent
                    )

                Data.Failure err ->
                    ( popError "Error moving node" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        ApiRespPutMirrorNode parent resp ->
            case resp of
                Data.Success _ ->
                    ( { model
                        | nodeOp = OpNone
                        , copyMove = CopyMoveNone
                        , nodes = List.map (expChildren parent) model.nodes
                      }
                    , refetchChildren model parent
                    )

                Data.Failure err ->
                    ( popError "Error mirroring node" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        ApiRespPutDuplicateNode parent resp ->
            case resp of
                Data.Success _ ->
                    ( { model
                        | nodeOp = OpNone
                        , copyMove = CopyMoveNone
                        , nodes = List.map (expChildren parent) model.nodes
                      }
                    , refetchChildren model parent
                    )

                Data.Failure err ->
                    ( popError "Error duplicating node" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        ApiRespPostNotificationNode resp ->
            case resp of
                Data.Success _ ->
                    ( { model | nodeOp = OpNone }, Effect.none )

                Data.Failure err ->
                    ( popError "Error messaging node" err model, Effect.none )

                _ ->
                    ( model, Effect.none )

        NatsEvent (Err err) ->
            ( popErrorStr ("Bad message from the connection: " ++ Decode.errorToString err) model
            , Effect.none
            )

        NatsEvent (Ok event) ->
            updateNats shared event model

        CopyNode feID id src desc ->
            ( { model
                | copyMove = Copy id src desc
                , nodeMsg =
                    Just
                        { feID = feID
                        , text = "Node copied\nclick paste in destination node"
                        , time = model.now
                        }
              }
            , Effect.fromCmd <| Port.out <| Port.encodeClipboard id
            )

        ClearClipboard ->
            ( { model | copyMove = CopyMoveNone }, Effect.none )

        UpdateNewPointType typ ->
            let
                addPoint =
                    model.addPoint

                addPointNew =
                    { addPoint | typ = typ }
            in
            ( { model | addPoint = addPointNew }, Effect.none )

        UpdateNewPointKey key ->
            let
                addPoint =
                    model.addPoint

                addPointNew =
                    { addPoint | key = key }
            in
            ( { model | addPoint = addPointNew }, Effect.none )

        UpdateCustomNodeType typ ->
            ( { model | customNodeType = typ }, Effect.none )

        ToggleRaw id ->
            let
                viewRaw =
                    case model.nodeEdit of
                        Just ne ->
                            if id == ne.feID then
                                not ne.viewRaw

                            else
                                True

                        Nothing ->
                            True
            in
            ( { model
                | nodeEdit =
                    if viewRaw then
                        Just
                            { feID = id
                            , points = []
                            , viewRaw = True
                            }

                    else
                        Nothing
              }
            , Effect.none
            )


{-| updateNats applies what the connection reports: connection state,
fetched subtrees, and live points.
-}
updateNats : Shared.Model -> Nats.Event -> Model -> ( Model, Effect Msg )
updateNats shared event model =
    case event of
        Nats.Connected c ->
            -- fetch every group afresh, and every expanded node, since
            -- points sent while the connection was down were missed
            let
                nodes =
                    List.filter (\t -> List.member (Tree.label t).anchor c.anchors) model.nodes

                fetches =
                    List.map (\g -> fetch { anchor = g, parent = "all", id = g, depth = 2 }) c.anchors
                        ++ List.map (\e -> fetchChildren e.anchor e.id) (NodeTree.expandedIDs nodes)
            in
            ( { model | connected = True, watching = [], nodes = nodes }
            , Effect.batch fetches
            )

        Nats.Disconnected ->
            ( { model | connected = False }, Effect.none )

        Nats.AuthFailed ->
            ( { model | error = Just "Signed Out" }
            , Effect.fromCmd <| Storage.signOut shared.storage
            )

        Nats.Nodes r ->
            let
                nodes =
                    if r.parent == "all" then
                        NodeTree.replaceAnchor r.anchor r.depth r.nodes model.nodes

                    else if r.id == "all" then
                        NodeTree.replaceChildren r.anchor r.parent r.depth r.nodes model.nodes

                    else
                        NodeTree.replaceChild r.anchor r.parent r.id r.depth r.nodes model.nodes
            in
            ( { model | nodes = NodeTree.finish nodes }, Effect.none )

        Nats.Points items ->
            let
                nodes =
                    List.foldl (\i n -> NodeTree.applyPoints i.nodeId i.points n) model.nodes items
            in
            ( { model | nodes = NodeTree.finish nodes }, Effect.none )

        Nats.EdgePoints items ->
            -- an edge the tree does not have is a new child of a node on
            -- screen: fetch it, with its children if the parent is open
            let
                ( nodes, fetches ) =
                    List.foldl
                        (\i ( n, f ) ->
                            let
                                ( n2, missing ) =
                                    NodeTree.applyEdgePoints i.nodeId i.parentId i.points n

                                deleted =
                                    Point.getBool i.points Point.typeTombstone ""
                            in
                            ( n2
                            , if deleted then
                                f

                              else
                                f
                                    ++ List.map
                                        (\m ->
                                            fetch
                                                { anchor = m.anchor
                                                , parent = m.parentID
                                                , id = i.nodeId
                                                , depth =
                                                    if m.expanded then
                                                        1

                                                    else
                                                        0
                                                }
                                        )
                                        missing
                            )
                        )
                        ( model.nodes, [] )
                        items
            in
            ( { model | nodes = NodeTree.finish nodes }, Effect.batch fetches )

        Nats.Error message ->
            ( popErrorStr message { model | nodes = NodeTree.clearLoading model.nodes }
            , Effect.none
            )


fetch : { anchor : String, parent : String, id : String, depth : Int } -> Effect Msg
fetch f =
    Effect.fromCmd <| Nats.send (Nats.Fetch f)


{-| fetchChildren asks for the children of a node, with their children,
so each child knows whether it can be expanded.
-}
fetchChildren : String -> String -> Effect Msg
fetchChildren anchor id =
    fetch { anchor = anchor, parent = id, id = "all", depth = 1 }


refetchChildren : Model -> Int -> Effect Msg
refetchChildren model feID =
    case NodeTree.findByFeID feID model.nodes of
        Just n ->
            fetchChildren n.anchor n.node.id

        Nothing ->
            Effect.none


sendPoints : Model -> String -> List Point -> Effect Msg
sendPoints model id points =
    case NodeTree.anchorOf id model.nodes of
        Just anchor ->
            Effect.fromCmd <| Nats.send (Nats.SendPoints { anchor = anchor, id = id, points = points })

        Nothing ->
            Effect.none


expChildren : Int -> Tree NodeView -> Tree NodeView
expChildren feID tree =
    Tree.map
        (\n ->
            if n.feID == feID then
                { n | expChildren = True }

            else
                n
        )
        tree


toggleExpDetail : List (Tree NodeView) -> Int -> List (Tree NodeView)
toggleExpDetail nodes feID =
    List.map
        (Tree.map
            (\n ->
                if n.feID == feID then
                    { n | expDetail = not n.expDetail }

                else
                    n
            )
        )
        nodes


popError : String -> Http.Error -> Model -> Model
popError desc err model =
    popErrorStr (desc ++ ": " ++ Data.errorToString err) model


popErrorStr : String -> Model -> Model
popErrorStr message model =
    { model | error = Just message, lastError = model.now }


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 1000 Tick
        , Nats.receive NatsEvent
        ]



-- VIEW


view : Auth.User -> Shared.Model -> Model -> View Msg
view _ shared model =
    { title = "SIOT"
    , attributes = []
    , element =
        UI.Layout.layout
            { onSignOut = SignOut
            , email = Maybe.map .email shared.storage.user
            , error = model.error
            , badge =
                if model.connected then
                    Nothing

                else
                    Just "connecting..."
            }
            (viewBody model)
    }


viewBody : Model -> Element Msg
viewBody model =
    column
        [ width fill, spacing 32 ]
        [ wrappedRow [ spacing 10 ] <|
            (el Style.h2 <| text "Nodes")
                :: (case model.copyMove of
                        CopyMoveNone ->
                            []

                        Copy id _ desc ->
                            [ Icon.clipboard
                            , el [ Font.italic ] <| text desc
                            , el [ Font.size 12 ] <| text <| "(" ++ id ++ ")"
                            , Button.x ClearClipboard
                            ]
                   )
        , viewNodes model
        ]


viewNodes : Model -> Element Msg
viewNodes model =
    column
        [ width fill
        , spacing 24
        ]
    <|
        let
            treeWithEdits =
                mergeNodesEdit model.nodes model.nodeEdit
        in
        List.concatMap
            (\t ->
                viewNode model Nothing (Tree.label t) [] 0
                    :: viewNodesHelp 1 model t
            )
            treeWithEdits


viewNodesHelp :
    Int
    -> Model
    -> Tree NodeView
    -> List (Element Msg)
viewNodesHelp depth model tree =
    let
        node =
            Tree.label tree

        children =
            if node.expChildren then
                Tree.children tree

            else
                []
    in
    List.foldl
        (\child ret ->
            let
                childNode =
                    Tree.label child

                tombstone =
                    NodeTree.isTombstone childNode.node
            in
            if not tombstone then
                let
                    viewChildren =
                        List.map Tree.label
                            (Tree.children child)
                in
                ret
                    ++ viewNode model (Just node) childNode viewChildren depth
                    :: viewNodesHelp (depth + 1) model child

            else
                ret
        )
        []
        children


viewNode : Model -> Maybe NodeView -> NodeView -> List NodeView -> Int -> Element Msg
viewNode model parent node children depth =
    let
        viewRaw =
            case model.nodeEdit of
                Just ne ->
                    ne.feID == node.feID && ne.viewRaw

                Nothing ->
                    False

        nodeView =
            if viewRaw then
                NodeRaw.view

            else
                case node.node.typ of
                    "user" ->
                        NodeUser.view

                    "group" ->
                        NodeGroup.view

                    "modbus" ->
                        NodeModbus.view

                    "modbusIo" ->
                        NodeModbusIO.view

                    "mqtt" ->
                        NodeMqtt.view

                    "mqttSub" ->
                        NodeMqttSub.view

                    "mqttDevice" ->
                        NodeMqttDevice.view

                    "sparkplugGroup" ->
                        NodeSparkplugGroup.view

                    "sparkplugNode" ->
                        NodeSparkplugNode.view

                    "sparkplugDevice" ->
                        NodeSparkplugDevice.view

                    "iio" ->
                        NodeIio.view

                    "iioChannel" ->
                        NodeIioChannel.view

                    "oneWire" ->
                        NodeOneWire.view

                    "oneWireIO" ->
                        NodeOneWireIO.view

                    "serialDev" ->
                        NodeSerialDev.view

                    "canBus" ->
                        NodeCanBus.view

                    "rule" ->
                        NodeRule.view

                    "condition" ->
                        NodeCondition.view

                    "action" ->
                        NodeAction.view

                    "actionInactive" ->
                        NodeAction.view

                    "device" ->
                        NodeDevice.view

                    "deviceCred" ->
                        NodeDeviceCred.view

                    "enrollToken" ->
                        NodeEnrollToken.view

                    "msgService" ->
                        NodeMessageService.view

                    "variable" ->
                        NodeVariable.view

                    "signalGenerator" ->
                        SignalGenerator.view

                    "gps" ->
                        NodeGps.view

                    "gpio" ->
                        NodeGpio.view

                    "file" ->
                        File.view

                    "sync" ->
                        NodeSync.view

                    "db" ->
                        NodeDb.view

                    "particle" ->
                        NodeParticle.view

                    "shelly" ->
                        NodeShelly.view

                    "shellyIo" ->
                        NodeShellyIO.view

                    "metrics" ->
                        NodeMetrics.view

                    "networkManager" ->
                        NodeNetworkManager.view

                    "ntp" ->
                        NodeNTP.view

                    "browser" ->
                        NodeBrowser.view

                    "networkManagerDevice" ->
                        NodeNetworkManagerDevice.view

                    "networkManagerConn" ->
                        NodeNetworkManagerConn.view

                    "update" ->
                        NodeUpdate.view

                    "provisioning" ->
                        NodeProvisioning.view

                    "provisioningFile" ->
                        NodeProvisioningFile.view

                    _ ->
                        NodeRaw.view

        background =
            if node.expDetail then
                Style.colors.pale

            else
                Style.colors.none

        alignButton =
            el [ alignTop, paddingEach { top = 10, right = 0, left = 0, bottom = 0 } ]
    in
    el
        [ width fill
        , paddingEach { top = 0, right = 0, bottom = 0, left = depth * 35 }
        , Form.onEnterEsc (ApiPostPoints node.node.id) DiscardNodeOp
        ]
    <|
        row [ spacing 6 ]
            [ alignButton <|
                if node.loading then
                    Icon.loader

                else if not node.hasChildren then
                    Icon.blank

                else if node.expChildren then
                    Button.arrowDown (ToggleExpChildren node.feID)

                else
                    Button.arrowRight (ToggleExpChildren node.feID)
            , alignButton <|
                Button.dot (ToggleExpDetail node.feID)
            , column
                [ spacing 6, padding 6, width fill, Background.color background ]
                [ viewIf (Node.edgeRole node.node == Node.EdgeRoleMirror) viewMirrorBadge
                , nodeView
                    { now = model.now
                    , zone = model.zone
                    , modified = node.mod
                    , parent = Maybe.map .node parent
                    , node = node.node
                    , children = children
                    , nodes = model.nodes
                    , expDetail = node.expDetail
                    , onEditNodePoint = EditNodePoint node.feID
                    , onUploadFile = UploadFile node.node.id
                    , onGenerateKey = GenerateKey node.node.id
                    , generatedToken = model.generatedToken
                    , copy = model.copyMove
                    , scratch = model.scratch
                    , onEditScratch = EditScratch
                    }
                , viewIf viewRaw <|
                    column [ spacing 10 ]
                        [ Input.text []
                            { onChange = UpdateNewPointType
                            , text = model.addPoint.typ
                            , placeholder = Nothing
                            , label = Input.labelLeft [] <| text "New point type:"
                            }
                        , Input.text []
                            { onChange = UpdateNewPointKey
                            , text = model.addPoint.key
                            , placeholder = Nothing
                            , label = Input.labelLeft [] <| text "New point key:"
                            }
                        ]
                , viewIf node.mod <|
                    Form.buttonRow <|
                        [ Form.button
                            { label = "save"
                            , color = colors.blue
                            , onPress = ApiPostPoints node.node.id
                            }
                        , Form.button
                            { label = "discard"
                            , color = colors.gray
                            , onPress = DiscardEdits
                            }
                        ]
                            ++ (if viewRaw then
                                    [ Form.button
                                        { label = "add point"
                                        , color = colors.darkgreen
                                        , onPress =
                                            let
                                                key =
                                                    if model.addPoint.key == "" then
                                                        "0"

                                                    else
                                                        model.addPoint.key
                                            in
                                            EditNodePoint node.feID
                                                [ Point model.addPoint.typ
                                                    key
                                                    model.now
                                                    0
                                                    0
                                                    ""
                                                    0
                                                ]
                                        }
                                    ]

                                else
                                    []
                               )
                , if node.expDetail then
                    let
                        viewNodeOps =
                            viewNodeOperations node msg

                        msg =
                            Maybe.andThen
                                (\m ->
                                    if m.feID == node.feID then
                                        Just m.text

                                    else
                                        Nothing
                                )
                                model.nodeMsg
                    in
                    case model.nodeOp of
                        OpNone ->
                            viewNodeOps

                        OpNodeToAdd add ->
                            if add.feID == node.feID then
                                viewAddNode model.customNodeType node add

                            else
                                viewNodeOps

                        OpNodeMessage m ->
                            if m.feID == node.feID then
                                viewMsgNode m

                            else
                                viewNodeOps

                        OpNodeDelete feID parentId ->
                            if feID == node.feID then
                                viewDeleteNode node.node parentId

                            else
                                viewNodeOps

                        OpNodePaste feID id ->
                            if feID == node.feID then
                                viewPasteNode model.nodes feID id node.node.typ model.copyMove

                            else
                                viewNodeOps

                  else
                    Element.none
                ]
            ]


nodeTypesThatHaveChildNodes : List String
nodeTypesThatHaveChildNodes =
    [ Node.typeDevice
    , Node.typeProvisioning
    , Node.typeGroup
    , Node.typeModbus
    , Node.typeOneWire
    , Node.typeIio
    , Node.typeMqtt
    , Node.typeSerialDev
    , Node.typeCanBus
    , Node.typeRule
    , Node.typeNetworkManager
    ]


viewNodeOperations : NodeView -> Maybe String -> Element Msg
viewNodeOperations node msg =
    let
        desc =
            Point.getBestDesc node.node.points

        showNodeAdd =
            List.member node.node.typ
                nodeTypesThatHaveChildNodes
    in
    column [ spacing 6 ]
        [ row [ spacing 6 ]
            [ viewIf showNodeAdd <|
                Button.plusCircle (AddNode node.feID node.node.id)
            , Button.message (MsgNode node.feID node.node.id node.node.parent)
            , Button.x (DeleteNode node.feID node.node.parent)
            , Button.copy (CopyNode node.feID node.node.id node.node.parent desc)
            , Button.clipboard (PasteNode node.feID node.node.id)
            , Button.list (ToggleRaw node.feID)
            ]
        , case msg of
            Just m ->
                text m

            Nothing ->
                Element.none
        ]


nodeDescUser : Element Msg
nodeDescUser =
    row [] [ Icon.user, text "User" ]


nodeDescGroup : Element Msg
nodeDescGroup =
    row [] [ Icon.users, text "Group" ]


nodeDescModbus : Element Msg
nodeDescModbus =
    row [] [ Icon.bus, text "Modbus" ]


nodeDescModbusIO : Element Msg
nodeDescModbusIO =
    row [] [ Icon.io, text "Modbus IO" ]


nodeDescSerialDev : Element Msg
nodeDescSerialDev =
    row [] [ Icon.serialDev, text "Serial Device" ]


nodeDescCanBus : Element Msg
nodeDescCanBus =
    row [] [ Icon.serialDev, text "CAN Bus" ]


nodeDescRule : Element Msg
nodeDescRule =
    row [] [ Icon.list, text "Rule" ]


nodeDescMsgService : Element Msg
nodeDescMsgService =
    row [] [ Icon.send, text "Messaging Service" ]


nodeDescDb : Element Msg
nodeDescDb =
    row [] [ Icon.database, text "Database" ]


nodeDescParticle : Element Msg
nodeDescParticle =
    row [] [ Icon.particle, text "Particle" ]


nodeDescShelly : Element Msg
nodeDescShelly =
    row [] [ Icon.shelly, text "Shelly" ]


nodeDescVariable : Element Msg
nodeDescVariable =
    row [] [ Icon.variable, text "Variable" ]


nodeDescSignalGenerator : Element Msg
nodeDescSignalGenerator =
    row [] [ Icon.activity, text "Signal Generator" ]


nodeDescFile : Element Msg
nodeDescFile =
    row [] [ Icon.file, text "File" ]


nodeDescSync : Element Msg
nodeDescSync =
    row [] [ Icon.sync, text "sync" ]


nodeDescDeviceCred : Element Msg
nodeDescDeviceCred =
    row [] [ Icon.key, text "Device credential" ]


nodeDescEnrollToken : Element Msg
nodeDescEnrollToken =
    row [] [ Icon.key, text "Enrollment token" ]


nodeDescCondition : Element Msg
nodeDescCondition =
    row [] [ Icon.check, text "Condition" ]


nodeDescAction : Element Msg
nodeDescAction =
    row [] [ Icon.trendingUp, text "Action (rule active)" ]


nodeDescActionInactive : Element Msg
nodeDescActionInactive =
    row [] [ Icon.trendingDown, text "Action (rule inactive)" ]


nodeDescMetrics : Element Msg
nodeDescMetrics =
    row [] [ Icon.barChart, text "Metrics" ]


nodeDescUpdate : Element Msg
nodeDescUpdate =
    row [] [ Icon.update, text "Update" ]


nodeDescNetworkManager : Element Msg
nodeDescNetworkManager =
    row [] [ Icon.network, text "Network Manager" ]


nodeDescNetworkManagerConn : Element Msg
nodeDescNetworkManagerConn =
    row [] [ Icon.cable, text "Connection" ]


nodeDescNTP : Element Msg
nodeDescNTP =
    row [] [ Icon.clock, text "NTP" ]


nodeDescBrowser : Element Msg
nodeDescBrowser =
    row [] [ Icon.globe, text "Browser" ]


nodeDescGps : Element Msg
nodeDescGps =
    row [] [ Icon.mapPin, text "GPS" ]


nodeDescGpio : Element Msg
nodeDescGpio =
    row [] [ Icon.io, text "GPIO" ]


nodeDescIio : Element Msg
nodeDescIio =
    row [] [ Icon.io, text "IIO (analog IO)" ]


nodeDescIioChannel : Element Msg
nodeDescIioChannel =
    row [] [ Icon.io, text "IIO Channel" ]


nodeDescMqtt : Element Msg
nodeDescMqtt =
    row [] [ Icon.mqtt, text "MQTT" ]


nodeDescMqttSub : Element Msg
nodeDescMqttSub =
    row [] [ Icon.topic, text "MQTT Subscription" ]


viewAddNode : String -> NodeView -> NodeToAdd -> Element Msg
viewAddNode customNodeType parent add =
    column [ spacing 10 ]
        [ Input.radio [ spacing 6 ]
            { onChange = SelectAddNodeType
            , selected = add.typ
            , label = Input.labelAbove [] (el [ padding 12 ] <| text "Select node type to add: ")
            , options =
                (if parent.node.typ == Node.typeDevice then
                    [ Input.option Node.typeUser nodeDescUser
                    , Input.option Node.typeGroup nodeDescGroup
                    , Input.option Node.typeRule nodeDescRule
                    , Input.option Node.typeNetworkManager nodeDescNetworkManager
                    , Input.option Node.typeNTP nodeDescNTP
                    , Input.option Node.typeBrowser nodeDescBrowser
                    , Input.option Node.typeModbus nodeDescModbus
                    , Input.option Node.typeSerialDev nodeDescSerialDev
                    , Input.option Node.typeCanBus nodeDescCanBus
                    , Input.option Node.typeGps nodeDescGps
                    , Input.option Node.typeGpio nodeDescGpio
                    , Input.option Node.typeIio nodeDescIio
                    , Input.option Node.typeMqtt nodeDescMqtt
                    , Input.option Node.typeMsgService nodeDescMsgService
                    , Input.option Node.typeDb nodeDescDb
                    , Input.option Node.typeParticle nodeDescParticle
                    , Input.option Node.typeShelly nodeDescShelly
                    , Input.option Node.typeVariable nodeDescVariable
                    , Input.option Node.typeSignalGenerator nodeDescSignalGenerator
                    , Input.option Node.typeFile nodeDescFile
                    , Input.option Node.typeSync nodeDescSync
                    , Input.option Node.typeDeviceCred nodeDescDeviceCred
                    , Input.option Node.typeEnrollToken nodeDescEnrollToken
                    , Input.option Node.typeMetrics nodeDescMetrics
                    , Input.option Node.typeUpdate nodeDescUpdate
                    ]

                 else
                    []
                )
                    ++ (if parent.node.typ == Node.typeGroup then
                            [ Input.option Node.typeUser nodeDescUser
                            , Input.option Node.typeGroup nodeDescGroup
                            , Input.option Node.typeRule nodeDescRule
                            , Input.option Node.typeModbus nodeDescModbus
                            , Input.option Node.typeSerialDev nodeDescSerialDev
                            , Input.option Node.typeCanBus nodeDescCanBus
                            , Input.option Node.typeGps nodeDescGps
                            , Input.option Node.typeGpio nodeDescGpio
                            , Input.option Node.typeIio nodeDescIio
                            , Input.option Node.typeMqtt nodeDescMqtt
                            , Input.option Node.typeMsgService nodeDescMsgService
                            , Input.option Node.typeDb nodeDescDb
                            , Input.option Node.typeParticle nodeDescParticle
                            , Input.option Node.typeShelly nodeDescShelly
                            , Input.option Node.typeVariable nodeDescVariable
                            , Input.option Node.typeSignalGenerator nodeDescSignalGenerator
                            , Input.option Node.typeFile nodeDescFile
                            ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeModbus then
                            [ Input.option Node.typeModbusIO nodeDescModbusIO ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeIio then
                            [ Input.option Node.typeIioChannel nodeDescIioChannel ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeMqtt then
                            [ Input.option Node.typeMqttSub nodeDescMqttSub ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeRule then
                            [ Input.option Node.typeCondition nodeDescCondition
                            , Input.option Node.typeAction nodeDescAction
                            , Input.option Node.typeActionInactive nodeDescActionInactive
                            ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeCanBus then
                            [ Input.option Node.typeFile nodeDescFile ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeNetworkManager then
                            [ Input.option Node.typeNetworkManagerConn nodeDescNetworkManagerConn ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeSerialDev then
                            [ Input.option Node.typeFile nodeDescFile ]

                        else
                            []
                       )
                    ++ (if parent.node.typ == Node.typeProvisioning then
                            [ Input.option Node.typeFile nodeDescFile ]

                        else
                            []
                       )
                    ++ [ Input.option "custom" <| text "Custom" ]
            }
        , viewIf (add.typ == Just "custom") <|
            Input.text
                []
                { onChange = UpdateCustomNodeType
                , text = customNodeType
                , placeholder = Nothing
                , label = Input.labelLeft [] <| text "Custom node type:"
                }
        , Form.buttonRow
            [ case add.typ of
                Just _ ->
                    Form.button
                        { label = "add"
                        , color = Style.colors.blue
                        , onPress = ApiPostAddNode parent.feID
                        }

                Nothing ->
                    Element.none
            , Form.button
                { label = "cancel"
                , color = Style.colors.gray
                , onPress = DiscardNodeOp
                }
            ]
        ]


viewMsgNode : NodeMessage -> Element Msg
viewMsgNode msg =
    el [ width fill, paddingEach { top = 10, right = 0, left = 0, bottom = 0 } ] <|
        column
            [ width fill, spacing 32 ]
            [ Input.multiline [ width fill ]
                { onChange = UpdateMsg
                , text = msg.message
                , placeholder = Nothing
                , label = Input.labelAbove [] <| text "Send message to users:"
                , spellcheck = True
                }
            , Form.buttonRow
                [ Form.button
                    { label = "send now"
                    , color = Style.colors.blue
                    , onPress = ApiPostNotificationNode
                    }
                , Form.button
                    { label = "cancel"
                    , color = Style.colors.gray
                    , onPress = DiscardNodeOp
                    }
                ]
            ]


{-| A mirror displays a node that lives somewhere else, and nothing runs here.
Without a mark, a mirrored sensor sitting in a group looks like a sensor that
has stopped reporting.
-}
viewMirrorBadge : Element Msg
viewMirrorBadge =
    el
        [ Background.color colors.ltblue
        , Font.size 12
        , paddingXY 6 2
        ]
    <|
        text "mirror"


viewDeleteNode : Node -> String -> Element Msg
viewDeleteNode node parent =
    let
        -- deleting the node where it lives takes its mirrors with it. No
        -- count is offered, because mirrors can sit in groups this user
        -- cannot see and a count from the visible tree would be wrong.
        question =
            case Node.edgeRole node of
                Node.EdgeRolePrimary ->
                    "Delete this node, and any mirrors of it?"

                Node.EdgeRoleMirror ->
                    "Remove this mirror? The node itself is not deleted."

                Node.EdgeRoleNone ->
                    "Delete this node?"
    in
    el [ paddingEach { top = 10, right = 0, left = 0, bottom = 0 } ] <|
        row []
            [ text question
            , Form.buttonRow
                [ Form.button
                    { label = "yes"
                    , color = colors.red
                    , onPress = ApiDelete node.id parent
                    }
                , Form.button
                    { label = "no"
                    , color = colors.gray
                    , onPress = DiscardNodeOp
                    }
                ]
            ]


viewPasteNode : List (Tree NodeView) -> Int -> String -> String -> CopyMove -> Element Msg
viewPasteNode nodes feID dest destType copyMove =
    let
        -- Some node types are found by walking down from their parent
        -- rather than from the tree root, so a modbusIo dropped into a
        -- group stops working quietly. Mirroring is what was wanted in
        -- that case, and it is all that is offered.
        ownerOf id =
            findNode nodes id
                |> Maybe.map (.typ >> Node.owningParentType)
                |> Maybe.withDefault ""

        cancelButton =
            Form.buttonRow
                [ Form.button
                    { label = "cancel"
                    , color = colors.gray
                    , onPress = DiscardNodeOp
                    }
                ]

        moveButton op =
            Form.button
                { label = "move"
                , color = colors.darkgreen
                , onPress = op
                }

        mirrorButton op =
            Form.button
                { label = "mirror"
                , color = colors.blue
                , onPress = op
                }

        duplicateButton op =
            Form.button
                { label = "duplicate"
                , color = colors.red
                , onPress = op
                }
    in
    el [ paddingEach { top = 10, right = 0, left = 0, bottom = 0 } ] <|
        case copyMove of
            CopyMoveNone ->
                row []
                    [ text "Select node to copy/move first"
                    , cancelButton
                    ]

            Copy id src desc ->
                row [] <|
                    if id == dest then
                        [ text "Can't move/copy node to itself"
                        , cancelButton
                        ]

                    else if src == dest then
                        [ text <| "Copy " ++ desc ++ " here?"
                        , Form.buttonRow
                            [ duplicateButton <| ApiPutDuplicateNode feID id src dest
                            , cancelButton
                            ]
                        ]

                    else
                        case ownerOf id of
                            "" ->
                                [ text <| "Copy " ++ desc ++ " here?"
                                , Form.buttonRow
                                    [ moveButton <| ApiPostMoveNode feID id src dest
                                    , mirrorButton <| ApiPutMirrorNode feID id src dest
                                    , duplicateButton <| ApiPutDuplicateNode feID id src dest
                                    , cancelButton
                                    ]
                                ]

                            owner ->
                                if owner == destType then
                                    [ text <| "Copy " ++ desc ++ " here?"
                                    , Form.buttonRow
                                        [ moveButton <| ApiPostMoveNode feID id src dest
                                        , mirrorButton <| ApiPutMirrorNode feID id src dest
                                        , duplicateButton <| ApiPutDuplicateNode feID id src dest
                                        , cancelButton
                                        ]
                                    ]

                                else
                                    [ text <|
                                        desc
                                            ++ " belongs under a "
                                            ++ owner
                                            ++ " node and stops working if moved elsewhere. Mirror it here instead?"
                                    , Form.buttonRow
                                        [ mirrorButton <| ApiPutMirrorNode feID id src dest
                                        , cancelButton
                                        ]
                                    ]


mergeNodesEdit : List (Tree NodeView) -> Maybe NodeEdit -> List (Tree NodeView)
mergeNodesEdit nodes nodeEdit =
    case nodeEdit of
        Just edit ->
            List.map
                (Tree.map
                    (\n ->
                        if edit.feID == n.feID then
                            let
                                node =
                                    n.node
                            in
                            { n
                                | mod = True
                                , node =
                                    { node
                                        | points =
                                            Point.updatePoints node.points edit.points
                                    }
                            }

                        else
                            { n | mod = False }
                    )
                )
                nodes

        Nothing ->
            List.map (Tree.map (\n -> { n | mod = False })) nodes
