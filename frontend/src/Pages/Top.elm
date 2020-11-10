module Pages.Top exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Api.Response exposing (Response)
import Browser.Navigation exposing (Key)
import Components.NodeGeneral as NodeGeneral
import Element exposing (..)
import Http
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import Task
import Time
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


type alias NodeEdit =
    { id : String
    , point : Point
    }


type alias Model =
    { key : Key
    , nodeEdit : Maybe NodeEdit
    , zone : Time.Zone
    , now : Time.Posix
    , nodes : List Node
    , auth : Auth
    , error : Maybe String
    }


defaultModel : Key -> Model
defaultModel key =
    Model
        key
        Nothing
        Time.utc
        (Time.millisToPosix 0)
        []
        { email = "", token = "", isRoot = False }
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
    | EditNodeDescription String String
    | DiscardEditedNodeDescription
    | ApiDelete String
    | ApiPostPoint String Point
    | ApiRespList (Data (List Node))
    | ApiRespDelete (Data Response)
    | ApiRespPostPoint (Data Response)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EditNodeDescription id description ->
            ( { model
                | nodeEdit =
                    Just
                        { id = id
                        , point = Point.newText "" Point.typeDescription description
                        }
              }
            , Cmd.none
            )

        ApiPostPoint id point ->
            let
                -- optimistically update nodes
                nodes =
                    List.map
                        (\d ->
                            if d.id == id then
                                { d | points = Point.updatePoint d.points point }

                            else
                                d
                        )
                        model.nodes
            in
            ( { model | nodeEdit = Nothing, nodes = nodes }
            , Node.postPoint
                { token = model.auth.token
                , id = id
                , point = point
                , onResponse = ApiRespPostPoint
                }
            )

        DiscardEditedNodeDescription ->
            ( { model | nodeEdit = Nothing }
            , Cmd.none
            )

        ApiDelete id ->
            -- optimistically update nodes
            let
                nodes =
                    List.filter (\d -> d.id /= id) model.nodes
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

        ApiRespList nodes ->
            case nodes of
                Data.Success n ->
                    ( { model | nodes = n }, Cmd.none )

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
        List.map
            (\n ->
                NodeGeneral.view
                    { isRoot = model.auth.isRoot
                    , now = model.now
                    , zone = model.zone
                    , modified = n.mod
                    , node = n.node
                    , onApiDelete = ApiDelete
                    , onEditNodeDescription = EditNodeDescription
                    , onApiPostPoint = ApiPostPoint
                    , onDiscardEditedNodeDescription = DiscardEditedNodeDescription
                    }
            )
        <|
            mergeNodeEdit model.nodes model.nodeEdit


type alias NodeMod =
    { node : Node
    , mod : Bool
    }


mergeNodeEdit : List Node -> Maybe NodeEdit -> List NodeMod
mergeNodeEdit nodes devConfigEdit =
    case devConfigEdit of
        Just edit ->
            List.map
                (\n ->
                    if edit.id == n.id then
                        { node =
                            { n | points = Point.updatePoint n.points edit.point }
                        , mod = True
                        }

                    else
                        { node = n, mod = False }
                )
                nodes

        Nothing ->
            List.map (\n -> { node = n, mod = False }) nodes
