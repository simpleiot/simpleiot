module Pages.Top exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Api.Response exposing (Response)
import Browser.Navigation exposing (Key)
import Components.NodeDevice as NodeDevice
import Components.NodeGroup as NodeGroup
import Components.NodeUser as NodeUser
import Element exposing (..)
import Element.Input as Input
import Http
import List.Extra
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import Task
import Time
import Tree exposing (Tree)
import Tree.Zipper as Zipper exposing (Zipper)
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style
import Utils.Route


page : Page Params Model Msg
page =
    Page.application
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        , save = save
        , load = load
        }



-- INIT


type alias Params =
    ()


type alias Model =
    { key : Key
    , nodeEdit : Maybe NodeEdit
    , zone : Time.Zone
    , now : Time.Posix
    , nodes : Maybe (Tree NodeView)
    , auth : Auth
    , error : Maybe String
    , addNode : Maybe NodeToAdd
    , moveNode : Maybe NodeMove
    }


type alias NodeView =
    { node : Node
    , hasChildren : Bool
    , expDetail : Bool
    , expChildren : Bool
    , mod : Bool
    }


type alias NodeEdit =
    { id : String
    , points : List Point
    }


type alias NodeToAdd =
    { typ : Maybe String
    , parent : String
    }


type alias NodeMove =
    { id : String
    , description : String
    , oldParent : String
    , newParent : Maybe String
    }


defaultModel : Key -> Model
defaultModel key =
    Model
        key
        Nothing
        Time.utc
        (Time.millisToPosix 0)
        Nothing
        { email = "", token = "", isRoot = False }
        Nothing
        Nothing
        Nothing


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared { key } =
    let
        model =
            defaultModel key
    in
    case shared.auth of
        Just auth ->
            ( { model | auth = auth }
            , Cmd.batch
                [ Task.perform Zone Time.here
                , Task.perform Tick Time.now
                , Node.list { onResponse = ApiRespList, token = auth.token }
                ]
            )

        Nothing ->
            -- this is not ever used as site is redirected at high levels to sign-in
            ( model
            , Utils.Route.navigate shared.key Route.SignIn
            )



-- UPDATE


type Msg
    = Tick Time.Posix
    | Zone Time.Zone
    | EditNodePoint String Point
    | ToggleExpChildren String
    | ToggleExpDetail String
    | DiscardEdits
    | AddNode String
    | DiscardAddNode
    | MoveNode String String
    | DiscardMoveNode
    | MoveNodeDescription String
    | SelectAddNodeType String
    | ApiDelete String
    | ApiPostPoints String
    | ApiPostAddNode
    | ApiPostMoveNode
    | ApiRespList (Data (List Node))
    | ApiRespDelete (Data Response)
    | ApiRespPostPoint (Data Response)
    | ApiRespPostAddNode (Data Response)
    | ApiRespPostMoveNode (Data Response)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EditNodePoint id point ->
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
                        { id = id
                        , points = Point.updatePoint editPoints point
                        }
              }
            , Cmd.none
            )

        ApiPostPoints id ->
            case model.nodes of
                Just nodes ->
                    case model.nodeEdit of
                        Just edit ->
                            let
                                -- optimistically update nodes
                                updatedNodes =
                                    Tree.map
                                        (\n ->
                                            if n.node.id == id then
                                                let
                                                    node =
                                                        n.node
                                                in
                                                { n
                                                    | node =
                                                        { node
                                                            | points = Point.updatePoints node.points edit.points
                                                        }
                                                }

                                            else
                                                n
                                        )
                                        nodes
                            in
                            ( { model | nodeEdit = Nothing, nodes = Just updatedNodes }
                            , Node.postPoints
                                { token = model.auth.token
                                , id = id
                                , points = edit.points
                                , onResponse = ApiRespPostPoint
                                }
                            )

                        Nothing ->
                            ( model, Cmd.none )

                Nothing ->
                    ( model, Cmd.none )

        DiscardEdits ->
            ( { model | nodeEdit = Nothing }
            , Cmd.none
            )

        ToggleExpChildren id ->
            let
                nodes =
                    model.nodes |> Maybe.map (toggleExpChildren id)
            in
            ( { model | nodes = nodes }, Cmd.none )

        ToggleExpDetail id ->
            let
                nodes =
                    model.nodes |> Maybe.map (toggleExpDetail id)
            in
            ( { model | nodes = nodes }, Cmd.none )

        AddNode id ->
            ( { model | addNode = Just { typ = Nothing, parent = id } }, Cmd.none )

        MoveNode id parent ->
            ( { model
                | moveNode =
                    Just
                        { id = id
                        , description = ""
                        , oldParent = parent
                        , newParent = Nothing
                        }
              }
            , Cmd.none
            )

        DiscardMoveNode ->
            ( { model | moveNode = Nothing }, Cmd.none )

        MoveNodeDescription desc ->
            let
                newId =
                    model.nodes
                        |> Maybe.andThen (findNode desc)
                        |> Maybe.map .node
                        |> Maybe.map .id

                moveNode =
                    model.moveNode
                        |> Maybe.map
                            (\mn ->
                                { mn
                                    | description = desc
                                    , newParent = newId
                                }
                            )
            in
            ( { model | moveNode = moveNode }, Cmd.none )

        SelectAddNodeType typ ->
            case model.addNode of
                Just add ->
                    ( { model | addNode = Just { add | typ = Just typ } }, Cmd.none )

                Nothing ->
                    ( model, Cmd.none )

        DiscardAddNode ->
            ( { model | addNode = Nothing }, Cmd.none )

        ApiPostAddNode ->
            -- FIXME optimistically update nodes
            ( { model | addNode = Nothing }
            , case model.addNode of
                Just addNode ->
                    case addNode.typ of
                        Just typ ->
                            Node.insert
                                { token = model.auth.token
                                , onResponse = ApiRespPostAddNode
                                , node =
                                    { id = ""
                                    , typ = typ
                                    , parent = addNode.parent
                                    , points = []
                                    }
                                }

                        Nothing ->
                            Cmd.none

                Nothing ->
                    Cmd.none
            )

        ApiPostMoveNode ->
            ( model
            , case model.moveNode of
                Just moveNode ->
                    case moveNode.newParent of
                        Just newParent ->
                            Node.move
                                { token = model.auth.token
                                , id = moveNode.id
                                , oldParent = moveNode.oldParent
                                , newParent = newParent
                                , onResponse = ApiRespPostMoveNode
                                }

                        Nothing ->
                            Cmd.none

                Nothing ->
                    Cmd.none
            )

        ApiDelete id ->
            -- optimistically update nodes
            let
                nodes =
                    -- FIXME Tree.filter (\d -> d.id /= id) model.nodes
                    model.nodes
            in
            ( { model | nodes = nodes }
            , Node.delete { token = model.auth.token, id = id, onResponse = ApiRespDelete }
            )

        Zone zone ->
            ( { model | zone = zone }, Cmd.none )

        Tick now ->
            ( { model | now = now }
            , updateNodes model
            )

        ApiRespList resp ->
            case resp of
                Data.Success nodes ->
                    let
                        maybeNew =
                            nodeListToTree nodes
                                |> Maybe.map populateHasChildren

                        treeMerged =
                            case ( model.nodes, maybeNew ) of
                                ( Just current, Just new ) ->
                                    Just <| mergeNodeTree current new

                                ( _, Just new ) ->
                                    Just new

                                ( Just current, _ ) ->
                                    Just current

                                _ ->
                                    Nothing
                    in
                    ( { model | nodes = treeMerged }, Cmd.none )

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
                        , Utils.Route.navigate model.key Route.SignIn
                        )

                    else
                        ( popError "Error getting nodes" err model
                        , Cmd.none
                        )

                _ ->
                    ( model, Cmd.none )

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
                    , updateNodes model
                    )

        ApiRespPostAddNode resp ->
            case resp of
                Data.Success _ ->
                    ( model
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

        ApiRespPostMoveNode resp ->
            case resp of
                Data.Success _ ->
                    ( model
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


mergeNodeTree : Tree NodeView -> Tree NodeView -> Tree NodeView
mergeNodeTree current new =
    let
        z =
            Zipper.fromTree current
    in
    Tree.map
        (\n ->
            case Zipper.findFromRoot (\o -> o.node.id == n.node.id) z of
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


toggleExpChildren : String -> Tree NodeView -> Tree NodeView
toggleExpChildren id tree =
    Tree.map
        (\n ->
            if n.node.id == id then
                { n | expChildren = not n.expChildren }

            else
                n
        )
        tree


toggleExpDetail : String -> Tree NodeView -> Tree NodeView
toggleExpDetail id tree =
    Tree.map
        (\n ->
            if n.node.id == id then
                { n | expDetail = not n.expDetail }

            else
                n
        )
        tree


findNode : String -> Tree NodeView -> Maybe NodeView
findNode desc tree =
    Zipper.findFromRoot
        (\n -> Node.description n.node == desc)
        (Zipper.fromTree tree)
        |> Maybe.map Zipper.label


nodeListToTree : List Node -> Maybe (Tree NodeView)
nodeListToTree nodes =
    List.Extra.find (\n -> n.parent == "") nodes
        |> Maybe.map (populateChildren nodes)



-- populateChildren takes a list of nodes with a parent field and converts
-- this into a tree


populateChildren : List Node -> Node -> Tree NodeView
populateChildren nodes root =
    Zipper.toTree <|
        populateChildrenHelp
            (Zipper.fromTree <| Tree.singleton (nodeToNodeView root))
            nodes


nodeToNodeView : Node -> NodeView
nodeToNodeView node =
    { node = node
    , hasChildren = False
    , expDetail = False
    , expChildren = False
    , mod = False
    }


populateChildrenHelp : Zipper NodeView -> List Node -> Zipper NodeView
populateChildrenHelp z nodes =
    case
        Zipper.forward
            (List.foldr
                (\n zCur ->
                    if (Zipper.label zCur).node.id == n.parent then
                        Zipper.mapTree
                            (\t ->
                                Tree.appendChild
                                    (Tree.singleton
                                        (nodeToNodeView n)
                                    )
                                    t
                            )
                            zCur

                    else
                        zCur
                )
                z
                nodes
            )
    of
        Just zMod ->
            populateChildrenHelp zMod nodes

        Nothing ->
            z


populateHasChildren : Tree NodeView -> Tree NodeView
populateHasChildren tree =
    let
        children =
            Tree.children tree

        hasChildren =
            List.length children > 0

        label =
            Tree.label tree

        node =
            { label | hasChildren = hasChildren }
    in
    tree
        |> Tree.replaceLabel node
        |> Tree.replaceChildren
            (List.map
                (\c -> populateHasChildren c)
                children
            )


popError : String -> Http.Error -> Model -> Model
popError desc err model =
    { model | error = Just (desc ++ ": " ++ Data.errorToString err) }


updateNodes : Model -> Cmd Msg
updateNodes model =
    Node.list { onResponse = ApiRespList, token = model.auth.token }


save : Model -> Shared.Model -> Shared.Model
save model shared =
    { shared
        | error =
            case model.error of
                Nothing ->
                    shared.error

                Just _ ->
                    model.error
        , lastError =
            case model.error of
                Nothing ->
                    shared.lastError

                Just _ ->
                    shared.now
    }


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load shared model =
    ( { model | key = shared.key, error = Nothing }, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ Time.every 5000 Tick
        ]



-- VIEW


view : Model -> Document Msg
view model =
    { title = "SIOT Nodes"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Nodes"
            , viewNodes model
            ]
        ]
    }


viewNodes : Model -> Element Msg
viewNodes model =
    column
        [ width fill
        , spacing 24
        ]
    <|
        case model.nodes of
            Just tree ->
                let
                    treeWithEdits =
                        mergeNodeEdit tree model.nodeEdit
                in
                viewNode model (Tree.label treeWithEdits) 0
                    :: viewNodesHelp 1 model treeWithEdits

            Nothing ->
                [ text "No nodes to display" ]


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
    List.foldr
        (\child ret ->
            ret
                ++ viewNode model (Tree.label child) depth
                :: viewNodesHelp (depth + 1) model child
        )
        []
        children


viewNode : Model -> NodeView -> Int -> Element Msg
viewNode model node depth =
    let
        nodeView =
            case node.node.typ of
                "user" ->
                    NodeUser.view

                "group" ->
                    NodeGroup.view

                _ ->
                    NodeDevice.view
    in
    el [ width fill, paddingEach { top = 0, right = 0, bottom = 0, left = depth * 35 } ] <|
        row [ spacing 6 ]
            [ el [ alignTop ] <|
                if not node.hasChildren then
                    Icon.blank

                else if node.expChildren then
                    Icon.arrowDown (ToggleExpChildren node.node.id)

                else
                    Icon.arrowRight (ToggleExpChildren node.node.id)
            , el [ alignTop ] <|
                if node.expDetail then
                    Icon.minimize (ToggleExpDetail node.node.id)

                else
                    Icon.maximize (ToggleExpDetail node.node.id)
            , column
                [ spacing 6, width fill ]
                [ nodeView
                    { isRoot = model.auth.isRoot
                    , now = model.now
                    , zone = model.zone
                    , modified = node.mod
                    , node = node.node
                    , expDetail = node.expDetail
                    , onApiDelete = ApiDelete
                    , onEditNodePoint = EditNodePoint
                    , onDiscardEdits = DiscardEdits
                    , onApiPostPoints = ApiPostPoints
                    }
                , case ( node.expDetail, model.addNode, model.moveNode ) of
                    ( False, _, _ ) ->
                        Element.none

                    ( True, Just add, _ ) ->
                        if add.parent == node.node.id then
                            viewAddNode add

                        else
                            viewNodeOperations node.node.id node.node.parent

                    ( True, _, Just move ) ->
                        if move.id == node.node.id then
                            viewMoveNode move

                        else
                            viewNodeOperations node.node.id node.node.parent

                    ( True, _, _ ) ->
                        viewNodeOperations node.node.id node.node.parent
                ]
            ]


viewNodeOperations : String -> String -> Element Msg
viewNodeOperations id parent =
    row [ spacing 6 ]
        [ Icon.plusCircle (AddNode id)
        , if parent /= "" then
            Icon.move (MoveNode id parent)

          else
            Element.none
        ]


viewMoveNode : NodeMove -> Element Msg
viewMoveNode move =
    column [ spacing 10 ]
        [ Input.text []
            { text = move.description
            , placeholder = Just <| Input.placeholder [] <| text "description"
            , label = Input.labelAbove [] <| text "New parent node: "
            , onChange = MoveNodeDescription
            }
        , Form.buttonRow
            [ case move.newParent of
                Just _ ->
                    Form.button
                        { label = "move"
                        , color = Style.colors.blue
                        , onPress = ApiPostMoveNode
                        }

                Nothing ->
                    Element.none
            , Form.button
                { label = "cancel"
                , color = Style.colors.gray
                , onPress = DiscardMoveNode
                }
            ]
        ]


viewAddNode : NodeToAdd -> Element Msg
viewAddNode add =
    column [ spacing 10 ]
        [ Input.radio [ spacing 6 ]
            { onChange = SelectAddNodeType
            , selected = add.typ
            , label = Input.labelAbove [] (el [ padding 12 ] <| text "Select node type to add: ")
            , options =
                [ Input.option Node.typeUser (text "User")
                , Input.option Node.typeGroup (text "Group")
                ]
            }
        , Form.buttonRow
            [ case add.typ of
                Just _ ->
                    Form.button
                        { label = "add"
                        , color = Style.colors.blue
                        , onPress = ApiPostAddNode
                        }

                Nothing ->
                    Element.none
            , Form.button
                { label = "cancel"
                , color = Style.colors.gray
                , onPress = DiscardAddNode
                }
            ]
        ]


mergeNodeEdit : Tree NodeView -> Maybe NodeEdit -> Tree NodeView
mergeNodeEdit nodes nodeEdit =
    case nodeEdit of
        Just edit ->
            Tree.map
                (\n ->
                    if edit.id == n.node.id then
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
                nodes

        Nothing ->
            Tree.map (\n -> { n | mod = False }) nodes
