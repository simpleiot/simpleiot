module Pages.Home_ exposing (Model, Msg, NodeEdit, NodeMsg, NodeOperation, page)

import Api.Data as Data exposing (Data)
import Api.Node as Node exposing (Node, NodeView)
import Api.Point as Point exposing (Point)
import Api.Port as Port
import Api.Response exposing (Response)
import Auth
import Components.NodeAction as NodeAction
import Components.NodeCanBus as NodeCanBus
import Components.NodeCondition as NodeCondition
import Components.NodeDb as NodeDb
import Components.NodeDevice as NodeDevice
import Components.NodeFile as File
import Components.NodeGroup as NodeGroup
import Components.NodeMessageService as NodeMessageService
import Components.NodeMetrics as NodeMetrics
import Components.NodeModbus as NodeModbus
import Components.NodeModbusIO as NodeModbusIO
import Components.NodeNTP as NodeNTP
import Components.NodeNetworkManager as NodeNetworkManager
import Components.NodeNetworkManagerConn as NodeNetworkManagerConn
import Components.NodeNetworkManagerDevice as NodeNetworkManagerDevice
import Components.NodeOneWire as NodeOneWire
import Components.NodeOneWireIO as NodeOneWireIO
import Components.NodeOptions exposing (CopyMove(..))
import Components.NodeParticle as NodeParticle
import Components.NodeRule as NodeRule
import Components.NodeSerialDev as NodeSerialDev
import Components.NodeShelly as NodeShelly
import Components.NodeShellyIO as NodeShellyIO
import Components.NodeSignalGenerator as SignalGenerator
import Components.NodeSync as NodeSync
import Components.NodeUnknown as NodeUnknown
import Components.NodeUser as NodeUser
import Components.NodeVariable as NodeVariable
import Dict
import Effect exposing (Effect)
import Element exposing (..)
import Element.Background as Background
import Element.Font as Font
import Element.Input as Input
import File
import File.Select
import Gen.Params.Home_ exposing (Params)
import Http
import List.Extra
import Page
import Request
import Shared
import Storage
import Task
import Time
import Tree exposing (Tree)
import Tree.Zipper as Zipper
import UI.Button as Button
import UI.Form as Form
import UI.Icon as Icon
import UI.Layout
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)
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
    , zone : Time.Zone
    , now : Time.Posix
    , nodes : List (Tree NodeView)
    , error : Maybe String
    , lastError : Time.Posix
    , nodeOp : NodeOperation
    , copyMove : CopyMove
    , nodeMsg : Maybe NodeMsg
    , token : String
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
    | OpNodeDelete Int String String
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
        Time.utc
        (Time.millisToPosix 0)
        []
        Nothing
        (Time.millisToPosix 0)
        OpNone
        CopyMoveNone
        Nothing
        ""


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
            , Node.list { onResponse = ApiRespList, token = token }
            ]
    )



-- UPDATE


type Msg
    = SignOut
    | Tick Time.Posix
    | Zone Time.Zone
    | EditNodePoint Int (List Point)
    | UploadFile String
    | UploadSelected String File.File
    | UploadContents String File.File String
    | ToggleExpChildren Int
    | ToggleExpDetail Int
    | DiscardNodeOp
    | DiscardEdits
    | AddNode Int String
    | MsgNode Int String String
    | PasteNode Int String
    | DeleteNode Int String String
    | UpdateMsg String
    | SelectAddNodeType String
    | ApiDelete String String
    | ApiPostPoints String
    | ApiPostAddNode Int
    | ApiPostMoveNode Int String String String
    | ApiPutMirrorNode Int String String
    | ApiPutDuplicateNode Int String String
    | ApiPostNotificationNode
    | ApiRespList (Data (List Node))
    | ApiRespDelete (Data Response)
    | ApiRespPostPoint (Data Response)
    | ApiRespPostAddNode Int (Data Response)
    | ApiRespPostMoveNode Int (Data Response)
    | ApiRespPutMirrorNode Int (Data Response)
    | ApiRespPutDuplicateNode Int (Data Response)
    | ApiRespPostNotificationNode (Data Response)
    | CopyNode Int String String String
    | ClearClipboard
    | ToggleRaw Int


update : Shared.Model -> Msg -> Model -> ( Model, Effect Msg )
update shared msg model =
    case msg of
        SignOut ->
            ( model, Effect.fromCmd <| Storage.signOut shared.storage )

        EditNodePoint feID points ->
            let
                editPoints =
                    case model.nodeEdit of
                        Just ne ->
                            ne.points

                        Nothing ->
                            []
            in
            ( { model
                | nodeEdit =
                    Just
                        { feID = feID
                        , points = Point.updatePoints editPoints points
                        , viewRaw = False
                        }
              }
            , Effect.none
            )

        UploadFile id ->
            ( model, Effect.fromCmd <| File.Select.file [ "" ] (UploadSelected id) )

        UploadSelected id file ->
            let
                uploadContents =
                    UploadContents id file
            in
            ( model, Effect.fromCmd <| Task.perform uploadContents (File.toString file) )

        UploadContents id file contents ->
            let
                pointName =
                    Point Point.typeName "0" model.now 0 (File.name file) 0

                pointData =
                    Point Point.typeData "0" model.now 0 contents 0
            in
            ( model
            , Effect.fromCmd <|
                Node.postPoints
                    { token = model.token
                    , id = id
                    , points = [ pointName, pointData ]
                    , onResponse = ApiRespPostPoint
                    }
            )

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
                    , Effect.fromCmd <|
                        Node.postPoints
                            { token = model.token
                            , id = id
                            , points = points
                            , onResponse = ApiRespPostPoint
                            }
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
            let
                nodes =
                    toggleExpChildren model.nodes feID
            in
            ( { model | nodes = nodes }, Effect.none )

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

        DeleteNode feID id parent ->
            ( { model | nodeOp = OpNodeDelete feID id parent }, Effect.none )

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
                                        , typ = typ
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

        ApiPutMirrorNode parent id dest ->
            ( model
            , Effect.fromCmd <|
                Node.copy
                    { token = model.token
                    , id = id
                    , newParent = dest
                    , duplicate = False
                    , onResponse = ApiRespPutMirrorNode parent
                    }
            )

        ApiPutDuplicateNode parent id dest ->
            ( model
            , Effect.fromCmd <|
                Node.copy
                    { token = model.token
                    , id = id
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
                                , parent = msgNode.parent
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
            , updateNodes model
            )

        ApiRespList resp ->
            case resp of
                Data.Success nodes ->
                    let
                        new =
                            nodes
                                |> nodeListToTrees
                                |> List.map (populateHasChildren "")
                                |> sortNodeTrees
                                |> populateFeID
                                |> mergeNodeTrees model.nodes
                    in
                    ( { model | nodes = new }, Effect.none )

                Data.Failure err ->
                    let
                        signOut =
                            case err of
                                Http.BadStatus code ->
                                    code == 401

                                _ ->
                                    False
                    in
                    if signOut then
                        ( { model | error = Just "Signed Out" }
                        , Effect.fromCmd <| Storage.signOut shared.storage
                        )

                    else
                        ( popError "Error getting nodes" err model
                        , Effect.none
                        )

                _ ->
                    ( model, Effect.none )

        ApiRespDelete resp ->
            case resp of
                Data.Success _ ->
                    ( model
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error deleting device" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , updateNodes model
                    )

        ApiRespPostPoint resp ->
            case resp of
                Data.Success _ ->
                    ( model
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error posting point" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , Effect.none
                    )

        ApiRespPostAddNode parentFeID resp ->
            case resp of
                Data.Success _ ->
                    ( { model | nodes = List.map (expChildren parentFeID) model.nodes }
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error adding node" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , updateNodes model
                    )

        ApiRespPostMoveNode parent resp ->
            case resp of
                Data.Success _ ->
                    let
                        nodes =
                            List.map (expChildren parent) model.nodes
                    in
                    ( { model | nodeOp = OpNone, copyMove = CopyMoveNone, nodes = nodes }
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error moving node" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , updateNodes model
                    )

        ApiRespPutMirrorNode parent resp ->
            case resp of
                Data.Success _ ->
                    let
                        nodes =
                            List.map (expChildren parent) model.nodes
                    in
                    ( { model | nodeOp = OpNone, copyMove = CopyMoveNone, nodes = nodes }
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error mirroring node" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , updateNodes model
                    )

        ApiRespPutDuplicateNode parent resp ->
            case resp of
                Data.Success _ ->
                    let
                        nodes =
                            List.map (expChildren parent) model.nodes
                    in
                    ( { model | nodeOp = OpNone, copyMove = CopyMoveNone, nodes = nodes }
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error duplicating node" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , updateNodes model
                    )

        ApiRespPostNotificationNode resp ->
            case resp of
                Data.Success _ ->
                    ( { model | nodeOp = OpNone }
                    , updateNodes model
                    )

                Data.Failure err ->
                    ( popError "Error messaging node" err model
                    , updateNodes model
                    )

                _ ->
                    ( model
                    , updateNodes model
                    )

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


mergeNodeTrees : List (Tree NodeView) -> List (Tree NodeView) -> List (Tree NodeView)
mergeNodeTrees current new =
    List.map
        (\n ->
            let
                newRootNode =
                    Tree.label n
            in
            case
                List.Extra.find
                    (\c ->
                        let
                            curRootNode =
                                Tree.label c
                        in
                        newRootNode.node.id == curRootNode.node.id && newRootNode.node.parent == curRootNode.node.parent
                    )
                    current
            of
                Just cur ->
                    mergeNodeTree cur n

                Nothing ->
                    n
        )
        new


mergeNodeTree : Tree NodeView -> Tree NodeView -> Tree NodeView
mergeNodeTree current new =
    let
        z =
            Zipper.fromTree current
    in
    Tree.map
        (\n ->
            case
                Zipper.findFromRoot
                    (\o ->
                        o.node.id
                            == n.node.id
                            && o.parentID
                            == n.parentID
                    )
                    z
            of
                Just found ->
                    let
                        l =
                            Zipper.label found
                    in
                    { n
                        | expChildren = l.expChildren
                        , expDetail = l.expDetail
                    }

                Nothing ->
                    n
        )
        new



-- FeID stands for front-end ID. This is required because we may
-- have some duplicate nodes in the data set, so we simply give each
-- one a unique ID while we are working with them in the frontend


populateFeID : List (Tree NodeView) -> List (Tree NodeView)
populateFeID trees =
    List.indexedMap
        (\i nodes ->
            Tree.indexedMap
                (\j n ->
                    { n | feID = i * 10000 + j }
                )
                nodes
        )
        trees


toggleExpChildren : List (Tree NodeView) -> Int -> List (Tree NodeView)
toggleExpChildren nodes feID =
    List.map
        (Tree.map
            (\n ->
                if n.feID == feID then
                    { n | expChildren = not n.expChildren }

                else
                    n
            )
        )
        nodes


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


nodeListToTrees : List Node -> List (Tree NodeView)
nodeListToTrees nodes =
    List.foldr
        (\n ret ->
            if n.parent == "root" then
                populateChildren nodes n :: ret

            else
                ret
        )
        []
        nodes


populateChildren : List Node -> Node -> Tree NodeView
populateChildren nodes root =
    Tree.replaceChildren (List.map (populateChildren nodes) (getChildren nodes root))
        (Tree.singleton <| nodeToNodeView root)


getChildren : List Node -> Node -> List Node
getChildren nodes parent =
    List.foldr
        (\n acc ->
            if n.parent == parent.id then
                n :: acc

            else
                acc
        )
        []
        nodes


nodeToNodeView : Node -> NodeView
nodeToNodeView node =
    { node = node
    , feID = 0
    , parentID = ""
    , hasChildren = False
    , expDetail = False
    , expChildren = False
    , mod = False
    }


populateHasChildren : String -> Tree NodeView -> Tree NodeView
populateHasChildren parentID tree =
    let
        children =
            Tree.children tree

        hasChildren =
            List.foldr
                (\child count ->
                    let
                        tombstone =
                            isTombstone (Tree.label child).node
                    in
                    if tombstone then
                        count

                    else
                        count + 1
                )
                0
                children
                > 0

        label =
            Tree.label tree

        node =
            { label
                | hasChildren = hasChildren
                , parentID = parentID
            }
    in
    tree
        |> Tree.replaceLabel node
        |> Tree.replaceChildren
            (List.map
                (\c -> populateHasChildren node.node.id c)
                children
            )


sortNodeTrees : List (Tree NodeView) -> List (Tree NodeView)
sortNodeTrees trees =
    List.sortWith nodeSort trees |> List.map sortNodeTree



-- sortNodeTree recursively sorts the children of the nodes
-- sort by type and then description


sortNodeTree : Tree NodeView -> Tree NodeView
sortNodeTree nodes =
    let
        children =
            Tree.children nodes

        childrenSorted =
            List.sortWith nodeSort children
    in
    Tree.tree (Tree.label nodes) (List.map sortNodeTree childrenSorted)



-- nodeCustomSortRules struct determines how we sort nodes in the UI


nodeCustomSortRules : Dict.Dict String String
nodeCustomSortRules =
    Dict.fromList
        [ ( Node.typeDevice, "A" )
        , ( Node.typeUser, "B" )
        , ( Node.typeGroup, "C" )
        , ( Node.typeModbus, "D" )
        , ( Node.typeRule, "E" )
        , ( Node.typeSignalGenerator, "F" )
        , ( Node.typeOneWire, "G" )
        , ( Node.typeCanBus, "H" )
        , ( Node.typeSerialDev, "I" )
        , ( Node.typeMsgService, "J" )
        , ( Node.typeFile, "K" )
        , ( Node.typeVariable, "L" )
        , ( Node.typeDb, "M" )
        , ( Node.typeMetrics, "N" )
        , ( Node.typeParticle, "O" )
        , ( Node.typeShelly, "P" )
        , ( Node.typeShellyIO, "Q" )
        , ( Node.typeNetworkManager, "R" )
        , ( Node.typeNTP, "S" )

        -- rule subnodes
        , ( Node.typeCondition, "A" )
        , ( Node.typeAction, "B" )
        , ( Node.typeActionInactive, "C" )
        , ( Node.typeNetworkManagerDevice, "D" )
        , ( Node.typeNetworkManagerConn, "E" )
        ]


nodeCustomSort : String -> String
nodeCustomSort t =
    case Dict.get t nodeCustomSortRules of
        Just s ->
            s

        Nothing ->
            t


nodeSort : Tree NodeView -> Tree NodeView -> Order
nodeSort a b =
    let
        aNode =
            Tree.label a

        bNode =
            Tree.label b

        aType =
            nodeCustomSort aNode.node.typ

        bType =
            nodeCustomSort bNode.node.typ
    in
    if aType /= bType then
        compare aType bType

    else
        let
            aDesc =
                String.toLower <| Point.getBestDesc aNode.node.points

            bDesc =
                String.toLower <| Point.getBestDesc bNode.node.points
        in
        if aDesc /= bDesc then
            compare aDesc bDesc

        else
            let
                aIndex =
                    Point.getValue aNode.node.points Point.typeIndex ""

                bIndex =
                    Point.getValue bNode.node.points Point.typeIndex ""
            in
            if aIndex /= bIndex then
                compare aIndex bIndex

            else
                let
                    aID =
                        Point.getText aNode.node.points Point.typeID ""

                    bID =
                        Point.getText bNode.node.points Point.typeID ""
                in
                compare aID bID


popError : String -> Http.Error -> Model -> Model
popError desc err model =
    { model | error = Just (desc ++ ": " ++ Data.errorToString err), lastError = model.now }


updateNodes : Model -> Effect Msg
updateNodes model =
    Effect.fromCmd <| Node.list { onResponse = ApiRespList, token = model.token }


subscriptions : Model -> Sub Msg
subscriptions _ =
    Time.every 4000 Tick



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
                    isTombstone childNode.node
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


isTombstone : Node -> Bool
isTombstone node =
    Point.getBool node.edgePoints Point.typeTombstone ""


viewNode : Model -> Maybe NodeView -> NodeView -> List NodeView -> Int -> Element Msg
viewNode model parent node children depth =
    let
        viewRaw =
            case model.nodeEdit of
                Just ne ->
                    ne.feID == node.feID && ne.viewRaw

                Nothing ->
                    False

        nodeViewType =
            case node.node.typ of
                "user" ->
                    NodeUser.view

                "group" ->
                    NodeGroup.view

                "modbus" ->
                    NodeModbus.view

                "modbusIo" ->
                    NodeModbusIO.view

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

                "msgService" ->
                    NodeMessageService.view

                "variable" ->
                    NodeVariable.view

                "signalGenerator" ->
                    SignalGenerator.view

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

                "networkManagerDevice" ->
                    NodeNetworkManagerDevice.view

                "networkManagerConn" ->
                    NodeNetworkManagerConn.view

                _ ->
                    NodeUnknown.view

        nodeView =
            if viewRaw then
                NodeUnknown.view

            else
                nodeViewType

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
                if not node.hasChildren then
                    Icon.blank

                else if node.expChildren then
                    Button.arrowDown (ToggleExpChildren node.feID)

                else
                    Button.arrowRight (ToggleExpChildren node.feID)
            , alignButton <|
                Button.dot (ToggleExpDetail node.feID)
            , column
                [ spacing 6, padding 6, width fill, Background.color background ]
                [ nodeView
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
                    , copy = model.copyMove
                    }
                , viewIf node.mod <|
                    Form.buttonRow
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
                                viewAddNode node add

                            else
                                viewNodeOps

                        OpNodeMessage m ->
                            if m.feID == node.feID then
                                viewMsgNode m

                            else
                                viewNodeOps

                        OpNodeDelete feID id parentId ->
                            if feID == node.feID then
                                viewDeleteNode id parentId

                            else
                                viewNodeOps

                        OpNodePaste feID id ->
                            if feID == node.feID then
                                viewPasteNode feID id model.copyMove

                            else
                                viewNodeOps

                  else
                    Element.none
                ]
            ]


nodeTypesThatHaveChildNodes : List String
nodeTypesThatHaveChildNodes =
    [ Node.typeDevice
    , Node.typeGroup
    , Node.typeModbus
    , Node.typeOneWire
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
            , Button.x (DeleteNode node.feID node.node.id node.node.parent)
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


nodeDescNetworkManager : Element Msg
nodeDescNetworkManager =
    row [] [ Icon.network, text "Network Manager" ]


nodeDescNetworkManagerConn : Element Msg
nodeDescNetworkManagerConn =
    row [] [ Icon.cable, text "Connection" ]


nodeDescNTP : Element Msg
nodeDescNTP =
    row [] [ Icon.clock, text "NTP" ]


viewAddNode : NodeView -> NodeToAdd -> Element Msg
viewAddNode parent add =
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
                    , Input.option Node.typeModbus nodeDescModbus
                    , Input.option Node.typeSerialDev nodeDescSerialDev
                    , Input.option Node.typeCanBus nodeDescCanBus
                    , Input.option Node.typeMsgService nodeDescMsgService
                    , Input.option Node.typeDb nodeDescDb
                    , Input.option Node.typeParticle nodeDescParticle
                    , Input.option Node.typeShelly nodeDescShelly
                    , Input.option Node.typeVariable nodeDescVariable
                    , Input.option Node.typeSignalGenerator nodeDescSignalGenerator
                    , Input.option Node.typeFile nodeDescFile
                    , Input.option Node.typeSync nodeDescSync
                    , Input.option Node.typeMetrics nodeDescMetrics
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


viewDeleteNode : String -> String -> Element Msg
viewDeleteNode id parent =
    el [ paddingEach { top = 10, right = 0, left = 0, bottom = 0 } ] <|
        row []
            [ text "Delete this node?"
            , Form.buttonRow
                [ Form.button
                    { label = "yes"
                    , color = colors.red
                    , onPress = ApiDelete id parent
                    }
                , Form.button
                    { label = "no"
                    , color = colors.gray
                    , onPress = DiscardNodeOp
                    }
                ]
            ]


viewPasteNode : Int -> String -> CopyMove -> Element Msg
viewPasteNode feID dest copyMove =
    let
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
                            [ duplicateButton <| ApiPutDuplicateNode feID id dest
                            , cancelButton
                            ]
                        ]

                    else
                        [ text <| "Copy " ++ desc ++ " here?"
                        , Form.buttonRow
                            [ moveButton <| ApiPostMoveNode feID id src dest
                            , mirrorButton <| ApiPutMirrorNode feID id dest
                            , duplicateButton <| ApiPutDuplicateNode feID id dest
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
