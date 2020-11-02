module Pages.Top exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Node as Node exposing (Node)
import Api.Point as Point exposing (Point)
import Api.Response exposing (Response)
import Browser.Navigation exposing (Key)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import Http
import Shared
import Spa.Document exposing (Document)
import Spa.Generated.Route as Route
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import Task
import Time
import UI.Icon as Icon
import UI.Style as Style exposing (colors, size)
import Utils.Duration as Duration
import Utils.Iso8601 as Iso8601
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
    , deviceEdit : Maybe NodeEdit
    , zone : Time.Zone
    , now : Time.Posix
    , devices : List Node
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
                | deviceEdit =
                    Just
                        { id = id
                        , point = Point.newText "" Point.typeDescription description
                        }
              }
            , Cmd.none
            )

        ApiPostPoint id point ->
            let
                -- optimistically update devices
                devices =
                    List.map
                        (\d ->
                            if d.id == id then
                                { d | points = Point.updatePoint d.points point }

                            else
                                d
                        )
                        model.devices
            in
            ( { model | deviceEdit = Nothing, devices = devices }
            , Node.postPoint
                { token = model.auth.token
                , id = id
                , point = point
                , onResponse = ApiRespPostPoint
                }
            )

        DiscardEditedNodeDescription ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            )

        ApiDelete id ->
            -- optimistically update devices
            let
                devices =
                    List.filter (\d -> d.id /= id) model.devices
            in
            ( { model | devices = devices }
            , Node.delete { token = model.auth.token, id = id, onResponse = ApiRespDelete }
            )

        Zone zone ->
            ( { model | zone = zone }, Cmd.none )

        Tick now ->
            ( { model | now = now }
            , updateNodes model
            )

        ApiRespList devices ->
            case devices of
                Data.Success d ->
                    ( { model | devices = d }, Cmd.none )

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
                        ( popError "Error getting devices" err model
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
            (\d ->
                viewNode model d.mod d.device
            )
        <|
            mergeNodeEdit model.devices model.deviceEdit


type alias NodeMod =
    { device : Node
    , mod : Bool
    }


mergeNodeEdit : List Node -> Maybe NodeEdit -> List NodeMod
mergeNodeEdit devices devConfigEdit =
    case devConfigEdit of
        Just edit ->
            List.map
                (\d ->
                    if edit.id == d.id then
                        { device =
                            { d | points = Point.updatePoint d.points edit.point }
                        , mod = True
                        }

                    else
                        { device = d, mod = False }
                )
                devices

        Nothing ->
            List.map (\d -> { device = d, mod = False }) devices


viewNode : Model -> Bool -> Node -> Element Msg
viewNode model modified device =
    let
        sysState =
            case Point.getPoint device.points "" Point.typeSysState 0 of
                Just point ->
                    round point.value

                Nothing ->
                    0

        sysStateIcon =
            case sysState of
                -- not sure why I can't use defines in Node.elm here
                1 ->
                    Icon.power

                2 ->
                    Icon.cloudOff

                3 ->
                    Icon.cloud

                _ ->
                    Element.none

        background =
            case sysState of
                3 ->
                    Style.colors.white

                _ ->
                    Style.colors.gray

        hwVersion =
            case Point.getPoint device.points "" Point.typeHwVersion 0 of
                Just point ->
                    "HW: " ++ point.text

                Nothing ->
                    ""

        osVersion =
            case Point.getPoint device.points "" Point.typeOSVersion 0 of
                Just point ->
                    "OS: " ++ point.text

                Nothing ->
                    ""

        appVersion =
            case Point.getPoint device.points "" Point.typeAppVersion 0 of
                Just point ->
                    "App: " ++ point.text

                Nothing ->
                    ""

        latestPointTime =
            case Point.getLatest device.points of
                Just point ->
                    point.time

                Nothing ->
                    Time.millisToPosix 0
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , Background.color background
        , spacing 6
        ]
        [ wrappedRow [ spacing 10 ]
            [ sysStateIcon
            , viewNodeId device.id
            , if model.auth.isRoot then
                Icon.x (ApiDelete device.id)

              else
                Element.none
            , Input.text
                [ Background.color background ]
                { onChange = \d -> EditNodeDescription device.id d
                , text = Node.description device
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , label = Input.labelHidden "device description"
                }
            , if modified then
                Icon.check
                    (ApiPostPoint device.id
                        { typ = Point.typeDescription
                        , id = ""
                        , index = 0
                        , time = model.now
                        , value = 0
                        , text = Node.description device
                        , min = 0
                        , max = 0
                        }
                    )

              else
                Element.none
            , if modified then
                Icon.x DiscardEditedNodeDescription

              else
                Element.none
            ]
        , viewPoints <| Point.filterSpecialPoints device.points
        , text ("Last update: " ++ Iso8601.toDateTimeString model.zone latestPointTime)
        , text
            ("Time since last update: "
                ++ Duration.toString
                    (Time.posixToMillis model.now
                        - Time.posixToMillis latestPointTime
                    )
            )
        , if hwVersion /= "" && osVersion /= "" && appVersion /= "" then
            text
                ("Version: "
                    ++ hwVersion
                    ++ " "
                    ++ osVersion
                    ++ " "
                    ++ appVersion
                )

          else
            Element.none
        ]


viewNodeId : String -> Element Msg
viewNodeId id =
    el
        [ padding 16
        , size.heading
        ]
    <|
        text id


viewPoints : List Point.Point -> Element Msg
viewPoints ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (Point.renderPoint >> text) ios
