module Pages.Top exposing (Flags, Model, Msg, page)

import Data.Device as D
import Data.Duration as Duration
import Data.Sample exposing (Sample, renderSample)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Farmation.Portal.Is as Is
import Global
import Iso8601
import Page exposing (Document, Page)
import Round
import Task
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style exposing (colors, size)


type alias Flags =
    ()


type alias DeviceEdit =
    { id : String
    , config : D.Config
    }


type alias SwUpdate =
    { id : String
    , version : String
    }


type alias SetTank =
    { id : String
    , level : Float
    }


type alias Model =
    { deviceEdit : Maybe DeviceEdit
    , zone : Time.Zone
    , now : Time.Posix

    -- IS mods
    , swUpdate : Maybe SwUpdate
    , setTank : Maybe SetTank
    }


type Msg
    = EditDeviceDescription String String
    | EditSwUpdateVersion String String
    | EditSetTank String String
    | PostConfig String D.Config
    | DiscardEditedDeviceDescription
    | DeleteDevice String
    | DeviceCancelCmd String
    | Tick Time.Posix
    | Zone Time.Zone
      -- IS messages
    | ISFillTank String
    | DeviceCmd String String String


page : Page Flags Model Msg
page =
    Page.component
        { init = init
        , update = update
        , subscriptions = subscriptions
        , view = view
        }


init : Global.Model -> Flags -> ( Model, Cmd Msg, Cmd Global.Msg )
init _ _ =
    ( Model Nothing Time.utc (Time.millisToPosix 0)
    , Cmd.batch [ Task.perform Zone Time.here, Task.perform Tick Time.now ]
    , Cmd.none
    )


update : Global.Model -> Msg -> Model -> ( Model, Cmd Msg, Cmd Global.Msg )
update global msg model =
    case msg of
        EditDeviceDescription id description ->
            ( { model | deviceEdit = Just { id = id, config = { description = description } } }
            , Cmd.none
            , Cmd.none
            )

        PostConfig id config ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            , Global.send <| Global.UpdateDeviceConfig id config
            )

        DiscardEditedDeviceDescription ->
            ( { model | deviceEdit = Nothing }
            , Cmd.none
            , Cmd.none
            )

        DeleteDevice id ->
            ( model, Cmd.none, Global.send <| Global.DeleteDevice id )

        DeviceCancelCmd id ->
            ( model, Cmd.none, Global.send <| Global.DeviceCancelCmd id )

        Zone zone ->
            ( { model | zone = zone }, Cmd.none, Cmd.none )

        Tick now ->
            ( { model | now = now }
            , Cmd.none
            , case global.auth of
                Global.SignedIn _ ->
                    Global.send Global.RequestDevices

                Global.SignedOut _ ->
                    Cmd.none
            )

        ISFillTank id ->
            ( model
            , Cmd.none
            , Global.send <|
                Global.UpdateDeviceCmd id { cmd = "fillTank", detail = "" }
            )

        DeviceCmd id cmd detail ->
            ( model
            , Cmd.none
            , Global.send <|
                Global.UpdateDeviceCmd id { cmd = cmd, detail = detail }
            )

        EditSwUpdateVersion id version ->
            ( { model | swUpdate = Just { id = id, version = version } }
            , Cmd.none
            , Cmd.none
            )

        EditSetTank id level ->
            let
                levelF =
                    if level == "" then
                        0

                    else
                        Maybe.withDefault
                            (Maybe.withDefault
                                { id = ""
                                , level = 0
                                }
                                model.setTank
                            ).level
                            (String.toFloat level)
            in
            ( { model | setTank = Just { id = id, level = levelF } }
            , Cmd.none
            , Cmd.none
            )


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.batch
        [ Time.every 5000 Tick
        ]


view : Global.Model -> Model -> Document Msg
view global model =
    { title = "IS Devices"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Devices"
            , case global.auth of
                Global.SignedIn sess ->
                    viewDevices sess.data.devices model sess.isRoot

                _ ->
                    el [ padding 16 ] <| text "Sign in to view your devices."
            ]
        ]
    }


viewDevices : List D.Device -> Model -> Bool -> Element Msg
viewDevices devices model isRoot =
    column
        [ width fill
        , spacing 24
        ]
    <|
        List.map
            (\d ->
                viewDevice model d.mod d.device isRoot
            )
        <|
            mergeDeviceEdit devices model.deviceEdit


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
                        { device = { d | config = edit.config }, mod = True }

                    else
                        { device = d, mod = False }
                )
                devices

        Nothing ->
            List.map (\d -> { device = d, mod = False }) devices


viewDevice : Model -> Bool -> D.Device -> Bool -> Element Msg
viewDevice model modified device isRoot =
    let
        sysStateIcon =
            case device.state.sysState of
                -- not sure who D.sysStatePowerOff does not work here ...
                1 ->
                    Icon.power

                2 ->
                    Icon.cloudOff

                3 ->
                    Icon.cloud

                _ ->
                    Element.none

        background =
            case device.state.sysState of
                3 ->
                    Style.colors.white

                _ ->
                    Style.colors.gray
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
                , text = device.config.description
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , label = Input.labelHidden "device description"
                }
            , if modified then
                Icon.check (PostConfig device.id device.config)

              else
                Element.none
            , if modified then
                Icon.x DiscardEditedDeviceDescription

              else
                Element.none
            ]
        , viewIoList device.state.ios
        , text ("Last update: " ++ Iso8601.toDateTimeString model.zone device.state.lastComm)
        , text
            ("Time since last update: "
                ++ Duration.toString
                    (Time.posixToMillis model.now
                        - Time.posixToMillis device.state.lastComm
                    )
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


viewIoList : List Sample -> Element Msg
viewIoList ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (renderSample >> text) ios


viewIS : D.Device -> Bool -> Bool -> Model -> Element Msg
viewIS device mod isRoot model =
    let
        -- following is just to make linter happy
        _ =
            viewDevice

        sysStateIcon =
            case device.state.sysState of
                -- not sure who D.sysStatePowerOff does not work here ...
                1 ->
                    Icon.power

                2 ->
                    Icon.cloudOff

                3 ->
                    Icon.cloud

                _ ->
                    Element.none

        background =
            case device.state.sysState of
                3 ->
                    Style.colors.white

                _ ->
                    Style.colors.gray

        is =
            Is.deviceToInjectorSentry device

        inputInj l =
            if is.inputInjector then
                l ++ [ image [] { src = "public/Injector.png", description = "injector" } ]

            else
                l

        inputWater l =
            if is.inputWaterOn then
                l ++ [ image [] { src = "public/WaterOn.png", description = "water on" } ]

            else
                l

        inputIrr l =
            if is.inputIrrigator then
                l ++ [ image [] { src = "public/Irrigator.png", description = "irrigator" } ]

            else
                l

        flow l =
            l ++ [ el Style.h3 <| text (Round.round 1 is.flowRate ++ " GPH") ]

        armed l =
            if is.armed then
                l ++ [ image [] { src = "public/Armed.png", description = "armed" } ]

            else
                l

        outputInj l =
            if is.outputInjector then
                l ++ [ image [] { src = "public/Injector.png", description = "injector" } ]

            else
                l

        outputShutdown l =
            if is.outputShutdown then
                l ++ [ image [] { src = "public/Shutdown.png", description = "shutdown" } ]

            else
                l

        statusElements =
            inputInj []
                |> inputWater
                |> inputIrr
                |> flow
                |> armed
                |> outputInj
                |> outputShutdown
    in
    column
        [ padding 20
        , spacing 20
        , width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , Background.color background
        ]
        [ wrappedRow [ spacing 10 ]
            [ sysStateIcon
            , el Style.h2 <| text device.id
            , if isRoot then
                Icon.x (DeleteDevice device.id)

              else
                Element.none
            , Input.text
                (Background.color background :: Style.h3)
                { label = Input.labelHidden "device description"
                , text = device.config.description
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , onChange = \d -> EditDeviceDescription device.id d
                }
            , if mod then
                Icon.check (PostConfig device.id device.config)

              else
                Element.none
            , if mod then
                Icon.x DiscardEditedDeviceDescription

              else
                Element.none
            ]
        , wrappedRow [ spacing 20 ] statusElements
        , if device.cmdPending then
            row [ spacing 20 ]
                [ el [ Font.color Style.colors.coral ] <| text "command pending ..."
                , Form.button "Cancel"
                    Style.colors.coral
                    (DeviceCancelCmd device.id)
                ]

          else
            Element.none
        , text ("Min Pressure: " ++ Round.round 0 is.pressureMin)
        , text ("Max Pressure: " ++ Round.round 0 is.pressureMax)
        , row [ spacing 20 ]
            [ text ("Tank Level: " ++ Round.round 0 is.currentTankVolume ++ " gal")
            , if device.state.sysState == 3 then
                Form.button "Fill"
                    Style.colors.blue
                    (ISFillTank device.id)

              else
                Element.none
            ]
        , if device.state.sysState == 3 then
            row [ spacing 20 ]
                [ Input.text []
                    { onChange = \l -> EditSetTank device.id l
                    , text =
                        case model.setTank of
                            Nothing ->
                                ""

                            Just setTank ->
                                if setTank.id == device.id then
                                    String.fromFloat setTank.level

                                else
                                    ""
                    , placeholder = Just <| Input.placeholder [] <| text "tank level"
                    , label = Input.labelLeft [ centerY ] <| text "Set tank level"
                    }
                , case model.setTank of
                    Nothing ->
                        Element.none

                    Just setTank ->
                        if setTank.id == device.id then
                            Form.button "Set now"
                                Style.colors.blue
                                (DeviceCmd
                                    setTank.id
                                    "setTankLevel"
                                    (String.fromFloat setTank.level)
                                )

                        else
                            Element.none
                ]

          else
            Element.none
        , text ("Tank Capacity: " ++ Round.round 0 is.tankCapacity ++ " gal")
        , text ("Flow Setpoint: " ++ Round.round 0 is.flowRateTarget)
        , text
            ("Flow Window: "
                ++ Round.round 0 is.flowWindowLow
                ++ " to "
                ++ Round.round 0 is.flowWindowHigh
            )
        , text ("Last communication (UTC): " ++ Iso8601.fromTime device.state.lastComm)
        , text
            ("Versions: hw: "
                ++ device.state.version.hw
                ++ ", os: "
                ++ device.state.version.os
                ++ ", app: "
                ++ device.state.version.app
            )
        , if isRoot && device.state.sysState == 3 then
            row [ spacing 20 ]
                [ Input.text []
                    { onChange = \v -> EditSwUpdateVersion device.id v
                    , text =
                        case model.swUpdate of
                            Nothing ->
                                ""

                            Just swUpdate ->
                                if swUpdate.id == device.id then
                                    swUpdate.version

                                else
                                    ""
                    , placeholder = Just <| Input.placeholder [] <| text "enter SW version"
                    , label = Input.labelLeft [ centerY ] <| text "SW Update"
                    }
                , case model.swUpdate of
                    Nothing ->
                        Element.none

                    Just swUpdate ->
                        if swUpdate.id == device.id then
                            Form.button "Update now"
                                Style.colors.blue
                                (DeviceCmd
                                    swUpdate.id
                                    "updateApp"
                                    ("https://storage.googleapis.com/farmation-update/is/is-app_"
                                        ++ swUpdate.version
                                        ++ ".xz"
                                    )
                                )

                        else
                            Element.none
                ]

          else
            Element.none
        ]
