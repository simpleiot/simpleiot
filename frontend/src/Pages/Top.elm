module Pages.Top exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Api.Response exposing (Response)
import Browser.Navigation exposing (Key)
import Components.NodeDevice as NodeDevice
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
import Tree.Zipper as Zipper
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
    , nodes : Maybe (Tree Node)
    , auth : Auth
    , error : Maybe String
    , addNode : Maybe NodeToAdd
    }


type alias NodeEdit =
    { id : String
    , points : List Point
    }


type alias NodeToAdd =
    { typ : Maybe String
    , parent : String
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
    | DiscardEdits
    | AddNode String
    | DiscardAddNode
    | SelectAddNodeType String
    | ApiDelete String
    | ApiPostPoints String
    | ApiPostAddNode
    | ApiRespList (Data (List Node))
    | ApiRespDelete (Data Response)
    | ApiRespPostPoint (Data Response)


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
                                            if n.id == id then
                                                { n | points = Point.updatePoints n.points edit.points }

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

        AddNode id ->
            ( { model | addNode = Just { typ = Nothing, parent = id } }, Cmd.none )

        SelectAddNodeType typ ->
            case model.addNode of
                Just add ->
                    ( { model | addNode = Just { add | typ = Just typ } }, Cmd.none )

                Nothing ->
                    ( model, Cmd.none )

        DiscardAddNode ->
            ( { model | addNode = Nothing }, Cmd.none )

        ApiPostAddNode ->
            ( model, Cmd.none )

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
                    ( { model | nodes = nodeListToTree nodes }, Cmd.none )

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


nodeListToTree : List Node -> Maybe (Tree Node)
nodeListToTree nodes =
    List.Extra.find (\n -> n.parent == "") nodes
        |> Maybe.map (populateChildren nodes)


populateChildren : List Node -> Node -> Tree Node
populateChildren nodes root =
    let
        z =
            Zipper.fromTree <| Tree.singleton root
    in
    Zipper.toTree <|
        List.foldr
            (\n zp ->
                if n.parent == "" then
                    -- skip the root node
                    zp

                else
                    -- find the parent child and add children
                    case Zipper.findFromRoot (\p -> p.id == n.parent) zp of
                        Just parent ->
                            Zipper.mapTree
                                (\t ->
                                    Tree.appendChild (Tree.singleton n) t
                                )
                                parent

                        Nothing ->
                            zp
            )
            z
            nodes


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
                viewNode model (Tree.label treeWithEdits)
                    :: viewNodes2Help 1 model treeWithEdits

            Nothing ->
                [ text "No nodes to display" ]


viewNodes2Help :
    Int
    -> Model
    -> Tree NodeMod
    -> List (Element Msg)
viewNodes2Help depth model tree =
    let
        children =
            Tree.children tree
    in
    List.foldr
        (\child ret ->
            ret
                ++ viewNode model (Tree.label child)
                :: viewNodes2Help (depth + 1) model child
        )
        []
        children


viewNode : Model -> NodeMod -> Element Msg
viewNode model node =
    let
        nodeView =
            case node.node.typ of
                "user" ->
                    NodeUser.view

                _ ->
                    NodeDevice.view
    in
    column [ spacing 6 ]
        [ nodeView
            { isRoot = model.auth.isRoot
            , now = model.now
            , zone = model.zone
            , modified = node.mod
            , node = node.node
            , onApiDelete = ApiDelete
            , onEditNodePoint = EditNodePoint
            , onDiscardEdits = DiscardEdits
            , onApiPostPoints = ApiPostPoints
            }
        , case model.addNode of
            Just add ->
                if add.parent == node.node.id then
                    viewAddNode add

                else
                    Icon.plusCircle (AddNode node.node.id)

            Nothing ->
                Icon.plusCircle (AddNode node.node.id)
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
                { label = "discard"
                , color = Style.colors.gray
                , onPress = DiscardAddNode
                }
            ]
        ]


type alias NodeMod =
    { node : Node
    , mod : Bool
    }


mergeNodeEdit : Tree Node -> Maybe NodeEdit -> Tree NodeMod
mergeNodeEdit nodes nodeEdit =
    case nodeEdit of
        Just edit ->
            Tree.map
                (\n ->
                    if edit.id == n.id then
                        { node =
                            { n | points = Point.updatePoints n.points edit.points }
                        , mod = True
                        }

                    else
                        { node = n, mod = False }
                )
                nodes

        Nothing ->
            Tree.map (\n -> { node = n, mod = False }) nodes
