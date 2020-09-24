module Pages.Top exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data as Data exposing (Data)
import Api.Device as Dev
import Api.Response exposing (Response)
import Browser.Navigation exposing (Key)
import Data.Duration as Duration
import Data.Iso8601 as Iso8601
import Data.Point as Point exposing (Point)
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


type alias DeviceEdit =
    { id : String
    , point : Point
    }


type alias Model =
    { key : Key
    , deviceEdit : Maybe DeviceEdit
    , zone : Time.Zone
    , now : Time.Posix
    , devices : List Dev.Device
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
                , Dev.list { onResponse = GotDevices, token = auth.token }
                ]
            )

        Nothing ->
            -- this is not ever used as site is redirected at high levels to sign-in
            ( model
            , Utils.Route.navigate shared.key Route.SignIn
            )



-- UPDATE


type Msg
    = EditDeviceDescription String String
    | PostPoint String Point
    | DiscardEditedDeviceDescription
    | DeleteDevice String
    | Tick Time.Posix
    | Zone Time.Zone
    | GotDevices (Data (List Dev.Device))
    | GotDeviceDeleted (Data Response)
    | GotPointPosted (Data Response)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EditDeviceDescription id description ->
            ( { model
                | deviceEdit =
                    Just
                        { id = id
                        , point = Point.newText "" Point.typeDescription description
                        }
              }
            , Cmd.none
            )

        PostPoint id point ->
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
            , Dev.postPoint
                { token = model.auth.token
                , id = id
                , point = point
                , onResponse = GotPointPosted
                }
            )

        DiscardEditedDeviceDescription ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            )

        DeleteDevice id ->
            -- optimistically update devices
            let
                devices =
                    List.filter (\d -> d.id /= id) model.devices
            in
            ( { model | devices = devices }
            , Dev.delete { token = model.auth.token, id = id, onResponse = GotDeviceDeleted }
            )

        Zone zone ->
            ( { model | zone = zone }, Cmd.none )

        Tick now ->
            ( { model | now = now }
            , updateDevices model
            )

        GotDevices devices ->
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

        GotDeviceDeleted resp ->
            case resp of
                Data.Success _ ->
                    ( model
                    , updateDevices model
                    )

                Data.Failure err ->
                    ( popError "Error deleting device" err model
                    , updateDevices model
                    )

                _ ->
                    ( model
                    , updateDevices model
                    )

        GotPointPosted resp ->
            case resp of
                Data.Success _ ->
                    ( model
                    , updateDevices model
                    )

                Data.Failure err ->
                    ( popError "Error posting point" err model
                    , updateDevices model
                    )

                _ ->
                    ( model
                    , updateDevices model
                    )


popError : String -> Http.Error -> Model -> Model
popError desc err model =
    { model | error = Just (desc ++ ": " ++ Data.errorToString err) }


updateDevices : Model -> Cmd Msg
updateDevices model =
    Dev.list { onResponse = GotDevices, token = model.auth.token }


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
    { title = "SIOT Devices"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Devices"
            , viewDevices model
            ]
        ]
    }


viewDevices : Model -> Element Msg
viewDevices model =
    column
        [ width fill
        , spacing 24
        ]
    <|
        List.map
            (\d ->
                viewDevice model d.mod d.device
            )
        <|
            mergeDeviceEdit model.devices model.deviceEdit


type alias DeviceMod =
    { device : Dev.Device
    , mod : Bool
    }


mergeDeviceEdit : List Dev.Device -> Maybe DeviceEdit -> List DeviceMod
mergeDeviceEdit devices devConfigEdit =
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


viewDevice : Model -> Bool -> Dev.Device -> Element Msg
viewDevice model modified device =
    let
        sysState =
            case Point.getPoint device.points "" Point.typeSysState 0 of
                Just point ->
                    round point.value

                Nothing ->
                    0

        sysStateIcon =
            case sysState of
                -- not sure why I can't use defines in Device.elm here
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
                    point.text

                Nothing ->
                    "?"

        osVersion =
            case Point.getPoint device.points "" Point.typeOSVersion 0 of
                Just point ->
                    point.text

                Nothing ->
                    "?"

        appVersion =
            case Point.getPoint device.points "" Point.typeAppVersion 0 of
                Just point ->
                    point.text

                Nothing ->
                    "?"

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
            , viewDeviceId device.id
            , if model.auth.isRoot then
                Icon.x (DeleteDevice device.id)

              else
                Element.none
            , Input.text
                [ Background.color background ]
                { onChange = \d -> EditDeviceDescription device.id d
                , text = Dev.description device
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , label = Input.labelHidden "device description"
                }
            , if modified then
                Icon.check
                    (PostPoint device.id
                        { typ = Point.typeDescription
                        , id = ""
                        , index = 0
                        , time = model.now
                        , value = 0
                        , text = Dev.description device
                        , min = 0
                        , max = 0
                        }
                    )

              else
                Element.none
            , if modified then
                Icon.x DiscardEditedDeviceDescription

              else
                Element.none
            ]
        , viewPoints device.points
        , text ("Last update: " ++ Iso8601.toDateTimeString model.zone latestPointTime)
        , text
            ("Time since last update: "
                ++ Duration.toString
                    (Time.posixToMillis model.now
                        - Time.posixToMillis latestPointTime
                    )
            )
        , text
            ("Version: HW: "
                ++ hwVersion
                ++ " OS: "
                ++ osVersion
                ++ " App: "
                ++ appVersion
            )
        ]


viewDeviceId : String -> Element Msg
viewDeviceId id =
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
