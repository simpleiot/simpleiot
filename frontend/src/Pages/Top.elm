module Pages.Top exposing (Model, Msg, Params, page)

import Api.Auth exposing (Auth)
import Api.Data exposing (Data)
import Api.Device as D
import Data.Duration as Duration
import Data.Iso8601 as Iso8601
import Data.Point as P
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Input as Input
import Shared
import Spa.Document exposing (Document)
import Spa.Page as Page exposing (Page)
import Spa.Url exposing (Url)
import Time
import UI.Icon as Icon
import UI.Style as Style exposing (colors, size)


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
    , point : P.Point
    }


type alias Model =
    { deviceEdit : Maybe DeviceEdit
    , zone : Time.Zone
    , now : Time.Posix
    , devices : Data (List D.Device)
    , auth : Maybe Auth
    }


defaultModel : Model
defaultModel =
    Model Nothing Time.utc (Time.millisToPosix 0) Api.Data.Loading Nothing


init : Shared.Model -> Url Params -> ( Model, Cmd Msg )
init shared { params } =
    ( { defaultModel | auth = shared.auth }
    , Cmd.none
    )



-- UPDATE


type Msg
    = EditDeviceDescription String String
    | PostPoint String P.Point
    | DiscardEditedDeviceDescription
    | DeleteDevice String
    | Tick Time.Posix
    | Zone Time.Zone
    | GotDevices (Data (List D.Device))


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        EditDeviceDescription id description ->
            ( { model
                | deviceEdit =
                    Just
                        { id = id
                        , point = P.newText "" P.typeDescription description
                        }
              }
            , Cmd.none
            )

        PostPoint id point ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
              --Global.send <| Global.UpdateDevicePoint id point
            )

        DiscardEditedDeviceDescription ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            )

        DeleteDevice id ->
            ( model
            , Cmd.none
              --Global.send <| Global.DeleteDevice id
            )

        Zone zone ->
            ( { model | zone = zone }, Cmd.none )

        Tick now ->
            ( { model | now = now }
            , Cmd.none
              --case global.auth of
              --    Global.SignedIn _ ->
              --        Global.send Global.RequestDevices
              --    Global.SignedOut _ ->
              --        Cmd.none
            )

        GotDevices devices ->
            ( { model | devices = devices }, Cmd.none )


save : Model -> Shared.Model -> Shared.Model
save model shared =
    shared


load : Shared.Model -> Model -> ( Model, Cmd Msg )
load shared model =
    ( model, Cmd.none )


subscriptions : Model -> Sub Msg
subscriptions model =
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
            , case model.auth of
                Just auth ->
                    viewDevices model auth

                _ ->
                    el [ padding 16 ] <| text "Sign in to view your devices."
            ]
        ]
    }


viewDevices : Model -> Auth -> Element Msg
viewDevices model auth =
    column
        [ width fill
        , spacing 24
        ]
    <|
        case model.devices of
            Api.Data.Loading ->
                [ text "Loading ..." ]

            Api.Data.Success devices ->
                List.map
                    (\d ->
                        viewDevice model d.mod d.device auth.isRoot
                    )
                <|
                    mergeDeviceEdit devices model.deviceEdit

            Api.Data.Failure _ ->
                [ text "Failed to load devices" ]

            Api.Data.NotAsked ->
                [ text "Not asked -- should never get this ..." ]


type alias DeviceMod =
    { device : D.Device
    , mod : Bool
    }


mergeDeviceEdit : List D.Device -> Maybe DeviceEdit -> List DeviceMod
mergeDeviceEdit devices devConfigEdit =
    case devConfigEdit of
        Just edit ->
            List.map
                (\d ->
                    if edit.id == d.id then
                        { device =
                            { d | points = P.updatePoint d.points edit.point }
                        , mod = True
                        }

                    else
                        { device = d, mod = False }
                )
                devices

        Nothing ->
            List.map (\d -> { device = d, mod = False }) devices


viewDevice : Model -> Bool -> D.Device -> Bool -> Element Msg
viewDevice model modified device isRoot =
    let
        sysState =
            case P.getPoint device.points "" P.typeSysState 0 of
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
            case P.getPoint device.points "" P.typeHwVersion 0 of
                Just point ->
                    point.text

                Nothing ->
                    "?"

        osVersion =
            case P.getPoint device.points "" P.typeOSVersion 0 of
                Just point ->
                    point.text

                Nothing ->
                    "?"

        appVersion =
            case P.getPoint device.points "" P.typeAppVersion 0 of
                Just point ->
                    point.text

                Nothing ->
                    "?"

        latestPointTime =
            case P.getLatest device.points of
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
            , if isRoot then
                Icon.x (DeleteDevice device.id)

              else
                Element.none
            , Input.text
                [ Background.color background ]
                { onChange = \d -> EditDeviceDescription device.id d
                , text = D.description device
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , label = Input.labelHidden "device description"
                }
            , if modified then
                Icon.check
                    (PostPoint device.id
                        { typ = P.typeDescription
                        , id = ""
                        , index = 0
                        , time = model.now
                        , value = 0
                        , text = D.description device
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


viewPoints : List P.Point -> Element Msg
viewPoints ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (P.renderPoint >> text) ios
