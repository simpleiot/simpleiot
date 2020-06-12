module Pages.Top exposing (Flags, Model, Msg, page)

import Data.Device as D
import Data.Sample exposing (Sample, renderSample)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import Farmation.Portal.Is as Is
import Global
import Iso8601
import Page exposing (Document, Page)
import Round
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


type alias Model =
    { deviceEdit : Maybe DeviceEdit
    }


type Msg
    = EditDeviceDescription String String
    | PostConfig String D.Config
    | DiscardEditedDeviceDescription
    | DeleteDevice String
    | DeviceCancelCmd String
    | Tick Time.Posix
    | ISFillTank String


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
    ( Model Nothing, Cmd.none, Global.send Global.RequestDevices )


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

        Tick _ ->
            ( model
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


subscriptions : Global.Model -> Model -> Sub Msg
subscriptions _ _ =
    Sub.batch
        [ Time.every 5000 Tick
        ]


view : Global.Model -> Model -> Document Msg
view global model =
    { title = "SIOT Devices"
    , body =
        [ column
            [ width fill, spacing 32 ]
            [ el Style.h2 <| text "Devices"
            , case global.auth of
                Global.SignedIn sess ->
                    viewDevices sess.data.devices model.deviceEdit sess.isRoot

                _ ->
                    el [ padding 16 ] <| text "Sign in to view your devices."
            ]
        ]
    }


viewDevices : List D.Device -> Maybe DeviceEdit -> Bool -> Element Msg
viewDevices devices deviceEdit isRoot =
    column
        [ width fill
        , spacing 24
        ]
    <|
        List.map
            (\dm -> viewIS dm.device dm.mod isRoot)
        <|
            mergeDeviceEdit devices deviceEdit


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


viewDevice : Bool -> D.Device -> Bool -> Element Msg
viewDevice mod device isRoot =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
        [ wrappedRow [ spacing 10 ]
            [ viewDeviceId device.id
            , if isRoot then
                Icon.x (DeleteDevice device.id)

              else
                Element.none
            , Input.text
                []
                { onChange = \d -> EditDeviceDescription device.id d
                , text = device.config.description
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , label = Input.labelHidden "device description"
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
        , viewIoList device.state.ios
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


viewIS : D.Device -> Bool -> Bool -> Element Msg
viewIS device mod isRoot =
    let
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
        ]
        [ wrappedRow [ spacing 10 ]
            [ el Style.h2 <| text device.id
            , if isRoot then
                Icon.x (DeleteDevice device.id)

              else
                Element.none
            , Input.text
                Style.h3
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
            , Form.button "Fill"
                Style.colors.blue
                (ISFillTank device.id)
            ]
        , text ("Tank Capacity: " ++ Round.round 0 is.tankCapacity ++ " gal")
        , text ("Flow: " ++ Round.round 0 is.flowRateTarget)
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
        ]
